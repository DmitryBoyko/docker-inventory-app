#Requires -Version 5.1
<#
.SYNOPSIS
  Инвентаризация Docker-контейнеров по compose-стекам (быстро, без N+1).

.DESCRIPTION
  Вызовы Docker:
    1) docker ps -a --size
    2) docker stats --no-stream
    3) docker inspect (все контейнеры)
    4) docker system df -v  (размеры томов)

  На контейнер: Service, Image, порты, IP, Networks, Volumes, CPU, Memory,
  NetIO, BlockIO, Disk (writable), Health, Restarts, Uptime, State.

  Под стеком: сумма RAM/CPU, топ по RAM, тома стека с размерами.

  Для parity harness (Go):
    .\scripts\docker-stack-inventory.ps1 -JsonOut .\parity-ps.json
#>
[CmdletBinding()]
param(
    # When set, write machine-readable parity JSON (schemaVersion=1) and skip Format-Table.
    [string]$JsonOut = ''
)

$ErrorActionPreference = 'Stop'
$script:ParityJsonMode = -not [string]::IsNullOrWhiteSpace($JsonOut)

function Convert-DockerSizeToBytes([string]$sizeText) {
    if ([string]::IsNullOrWhiteSpace($sizeText)) { return [int64]0 }
    $t = $sizeText.Trim()
    # "10.66MiB", "12.1GiB", "81.9kB", "11.09GB", "525.7kB"
    if ($t -match '^([0-9]+(?:\.[0-9]+)?)\s*([kKmMgGtT]i?[bB]|[bB])$') {
        $val  = [double]$Matches[1]
        $unit = $Matches[2]
        switch -Regex ($unit) {
            '^[bB]$'     { return [int64]$val }
            '^[kK][bB]$' { return [int64]($val * 1000) }
            '^[mM][bB]$' { return [int64]($val * 1000 * 1000) }
            '^[gG][bB]$' { return [int64]($val * 1000 * 1000 * 1000) }
            '^[tT][bB]$' { return [int64]($val * 1000L * 1000 * 1000 * 1000) }
            '^[kK]i[bB]$' { return [int64]($val * 1KB) }
            '^[mM]i[bB]$' { return [int64]($val * 1MB) }
            '^[gG]i[bB]$' { return [int64]($val * 1GB) }
            '^[tT]i[bB]$' { return [int64]($val * 1TB) }
            default { return [int64]$val }
        }
    }
    return [int64]0
}

function Format-Bytes([int64]$bytes) {
    if ($null -eq $bytes) { $bytes = 0 }
    if ($bytes -ge 1GB) { return ('{0:N2} GB' -f ($bytes / 1GB)) }
    if ($bytes -ge 1MB) { return ('{0:N2} MB' -f ($bytes / 1MB)) }
    if ($bytes -ge 1KB) { return ('{0:N2} KB' -f ($bytes / 1KB)) }
    return "$bytes B"
}

function Format-VolumeName([string]$name) {
    if ($name -match '^[a-f0-9]{64}$') {
        return ('anon:{0}...' -f $name.Substring(0, 12))
    }
    return $name
}

function Get-ShortImage([string]$image) {
    if ([string]::IsNullOrWhiteSpace($image)) { return '-' }
    # sha256:... -> short
    if ($image -match '^sha256:([a-f0-9]{12})') { return ('sha256:{0}...' -f $Matches[1]) }
    return $image
}

function Split-DockerPorts([string]$ports) {
    $extMap   = [ordered]@{}
    $internal = [System.Collections.Generic.List[string]]::new()
    $exposures = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)

    if ([string]::IsNullOrWhiteSpace($ports)) {
        return [pscustomobject]@{ External = '-'; Internal = '-'; Exposures = @() }
    }

    foreach ($part in ($ports -split ',\s*')) {
        $p = $part.Trim()
        if (-not $p) { continue }

        if ($p -match '->') {
            if ($p -match '^((?:\[[^\]]+\]|[^:]+)):(\d+)->(.+)$') {
                $hostIp = $Matches[1]
                $key    = "$($Matches[2])->$($Matches[3])"
                if (-not $extMap.Contains($key)) {
                    $extMap[$key] = [System.Collections.Generic.List[string]]::new()
                }
                if (-not $extMap[$key].Contains($hostIp)) {
                    [void]$extMap[$key].Add($hostIp)
                }
            } else {
                if (-not $extMap.Contains($p)) {
                    $extMap[$p] = [System.Collections.Generic.List[string]]::new()
                }
            }
        } else {
            if (-not $internal.Contains($p)) { [void]$internal.Add($p) }
            [void]$exposures.Add('internal')
        }
    }

    $externalParts = foreach ($key in $extMap.Keys) {
        $ips = @($extMap[$key])
        if ($ips.Count -eq 0) { $key; continue }

        $allIfaces = ($ips -contains '0.0.0.0') -or ($ips -contains '*') -or ($ips -contains '[::]')
        $onlyLocal = (@($ips | Where-Object { $_ -in @('127.0.0.1', '[::1]') }).Count -eq $ips.Count)
        $extra     = @($ips | Where-Object { $_ -notin @('0.0.0.0', '*', '[::]', '127.0.0.1', '[::1]') })

        if ($allIfaces -and $extra.Count -eq 0 -and -not $onlyLocal) {
            [void]$exposures.Add('public')
            "*:${key} [наружу]"
        } elseif ($onlyLocal) {
            [void]$exposures.Add('localhost')
            "127.0.0.1:${key} [localhost]"
        } else {
            [void]$exposures.Add('specific')
            ($ips | ForEach-Object { "${_}:${key}" }) -join '; '
        }
    }

    [pscustomobject]@{
        External   = if ($externalParts) { ($externalParts -join ' | ') } else { '-' }
        Internal   = if ($internal.Count) { ($internal -join '; ') } else { '-' }
        Exposures  = @($exposures | Sort-Object)
    }
}

function Normalize-ParityHealth([string]$health) {
    $h = ([string]$health).Trim().ToLowerInvariant()
    if ([string]::IsNullOrWhiteSpace($h) -or $h -eq '-') { return 'none' }
    if ($h -match 'unhealthy') { return 'unhealthy' }
    if ($h -match 'healthy') { return 'healthy' }
    if ($h -match 'starting') { return 'starting' }
    return 'unknown'
}

function Get-VolumeSizeMap {
    $map = @{}
    $inVolumes = $false
    foreach ($line in (docker system df -v)) {
        $t = [string]$line
        if ($t -match '^\s*VOLUME NAME\s+LINKS\s+SIZE') {
            $inVolumes = $true
            continue
        }
        if ($inVolumes) {
            if ([string]::IsNullOrWhiteSpace($t)) { break }
            if ($t -match 'Build cache') { break }
            # NAME  LINKS  SIZE  (SIZE may be "11.09GB" or "525.7kB")
            if ($t -match '^\s*(\S+)\s+(\d+)\s+(\S+)\s*$') {
                $map[$Matches[1]] = [pscustomobject]@{
                    Name  = $Matches[1]
                    Links = [int]$Matches[2]
                    Size  = $Matches[3]
                    Bytes = (Convert-DockerSizeToBytes $Matches[3])
                }
            }
        }
    }
    return $map
}

function Show-Think {
    param(
        [string]$Status,
        [int]$PercentComplete = 0
    )
    Write-Progress -Activity 'Сбор данных Docker' -Status $Status -PercentComplete $PercentComplete
}

# --- 1) Containers + SizeRw ---
Show-Think 'Контейнеры и размеры...' 5
$dockerJson = @(
    docker ps -a --size --format '{{json .}}' |
        ForEach-Object { $_ | ConvertFrom-Json }
)

# --- 2) Stats (running only) ---
Show-Think 'Память и CPU (stats)...' 30
$allStats = @{}
docker stats --no-stream --format '{{json .}}' | ForEach-Object {
    $s = $_ | ConvertFrom-Json
    if ($s.ID) { $allStats[$s.ID] = $s }
}

# --- 3) Inspect all ---
Show-Think 'Сеть, тома, health (inspect)...' 55
$inspectById = @{}
$ids = @(docker ps -aq)
if ($ids.Count -gt 0) {
    $inspected = docker inspect $ids | ConvertFrom-Json
    if ($inspected -isnot [System.Array]) { $inspected = @($inspected) }
    foreach ($c in $inspected) {
        $short = $c.Id.Substring(0, [Math]::Min(12, $c.Id.Length))
        $volNames = @(
            $c.Mounts |
                Where-Object { $_.Type -eq 'volume' -and $_.Name } |
                ForEach-Object { $_.Name }
        )
        $nets = @()
        $ips  = @()
        if ($c.NetworkSettings -and $c.NetworkSettings.Networks) {
            foreach ($prop in $c.NetworkSettings.Networks.PSObject.Properties) {
                $nets += $prop.Name
                $ip = $prop.Value.IPAddress
                if ($ip) { $ips += "$ip ($($prop.Name))" }
            }
        }
        $health = '-'
        if ($c.State.Health -and $c.State.Health.Status) {
            $health = [string]$c.State.Health.Status
        }

        $inspectById[$short] = [pscustomobject]@{
            Volumes     = $volNames
            Networks    = $nets
            IPs         = $ips
            Restarts    = [int]$c.RestartCount
            Health      = $health
        }
    }
}

# --- 4) Volume sizes ---
Show-Think 'Размеры томов...' 75
$volumeSizeMap = Get-VolumeSizeMap

Show-Think 'Обработка...' 90
$processedData = foreach ($item in $dockerJson) {
    $id = $item.ID

    $stackName = 'standalone'
    $serviceName = '-'
    if ($item.Labels -match 'com\.docker\.compose\.project=([^,]+)') {
        $stackName = $Matches[1]
    }
    if ($item.Labels -match 'com\.docker\.compose\.service=([^,]+)') {
        $serviceName = $Matches[1]
    }

    $stat = $allStats[$id]
    $memText = 'Stopped'
    $memBytes = [int64]0
    $cpuText = '-'
    $cpuVal = 0.0
    $netIO = '-'
    $blockIO = '-'
    if ($stat) {
        $memText = (($stat.MemUsage -split '\s*/\s*')[0]).Trim()
        $memBytes = Convert-DockerSizeToBytes $memText
        $cpuText = [string]$stat.CPUPerc
        if ($cpuText -match '([0-9]+(?:\.[0-9]+)?)') { $cpuVal = [double]$Matches[1] }
        $netIO = [string]$stat.NetIO
        $blockIO = [string]$stat.BlockIO
    }

    $diskRaw    = if ($item.Size) { [string]$item.Size } else { '0B' }
    $diskString = ($diskRaw -split '\s*\(virtual')[0].Trim()
    $ports      = Split-DockerPorts ([string]$item.Ports)

    $info = $inspectById[$id]
    $volList = @()
    $volDisplay = '-'
    $netsDisplay = '-'
    $ipDisplay = '-'
    $restarts = 0
    $health = '-'
    if ($info) {
        $volList = @($info.Volumes)
        if ($volList.Count) {
            $volDisplay = ($volList | ForEach-Object { Format-VolumeName $_ }) -join ', '
        }
        if ($info.Networks.Count) { $netsDisplay = ($info.Networks -join ', ') }
        if ($info.IPs.Count) { $ipDisplay = ($info.IPs -join '; ') }
        $restarts = $info.Restarts
        if ($info.Health -and $info.Health -ne '-') {
            $health = $info.Health
        }
    }
    # fallback health from Status text: Up 3 minutes (healthy)
    if ($health -eq '-' -and $item.Status -match '\(([^)]+)\)') {
        $maybe = $Matches[1]
        if ($maybe -match '^(healthy|unhealthy|health: starting|starting)$') {
            $health = $maybe
        }
    }

    $uptime = if ($item.RunningFor) { [string]$item.RunningFor } else { '-' }
    if ($item.State -eq 'exited' -or $item.State -eq 'created') {
        $uptime = [string]$item.Status
    }

    [pscustomobject]@{
        Stack       = $stackName
        Container   = $item.Names
        IdShort     = $id
        Service     = $serviceName
        Image       = (Get-ShortImage ([string]$item.Image))
        External    = $ports.External
        Internal    = $ports.Internal
        PortExposures = @($ports.Exposures)
        IP          = $ipDisplay
        Networks    = $netsDisplay
        Volumes     = $volDisplay
        VolumeNames = $volList
        CPU         = $cpuText
        CPUVal      = $cpuVal
        Memory      = $memText
        MemBytes    = $memBytes
        NetIO       = $netIO
        BlockIO     = $blockIO
        Disk        = $diskString
        DiskBytes   = (Convert-DockerSizeToBytes $diskString)
        Health      = $health
        Restarts    = $restarts
        Uptime      = $uptime
        State       = $item.State
    }
}

Write-Progress -Activity 'Сбор данных Docker' -Completed

if ($script:ParityJsonMode) {
    $containerRows = @(
        $processedData | ForEach-Object {
            [ordered]@{
                idShort            = [string]$_.IdShort
                name               = [string]$_.Container
                stack              = [string]$_.Stack
                service            = [string]$_.Service
                state              = [string]$_.State
                health             = (Normalize-ParityHealth ([string]$_.Health))
                restartCount       = [int]$_.Restarts
                writableLayerBytes = [int64]$_.DiskBytes
                volumeNames        = @($_.VolumeNames | Where-Object { $_ } | Sort-Object -Unique)
                cpuPercent         = [double]$_.CPUVal
                memoryBytes        = [int64]$_.MemBytes
                portExposures      = @($_.PortExposures)
            }
        }
    )

    $stackRows = @(
        $processedData | Group-Object Stack | Sort-Object Name | ForEach-Object {
            $g = $_.Group
            $volNames = @(
                $g | ForEach-Object { $_.VolumeNames } | Where-Object { $_ } | Select-Object -Unique | Sort-Object
            )
            $volBytes = [int64]0
            foreach ($vn in $volNames) {
                if ($volumeSizeMap.ContainsKey($vn)) { $volBytes += [int64]$volumeSizeMap[$vn].Bytes }
            }
            $wl = [int64]($g | Measure-Object DiskBytes -Sum).Sum
            $mem = [int64]($g | Where-Object { $_.State -eq 'running' } | Measure-Object MemBytes -Sum).Sum
            $cpu = [double]($g | Where-Object { $_.State -eq 'running' } | Measure-Object CPUVal -Sum).Sum
            [ordered]@{
                name               = [string]$_.Name
                containerCount     = [int]$g.Count
                runningCount       = [int]@($g | Where-Object { $_.State -eq 'running' }).Count
                unhealthyCount     = [int]@($g | Where-Object { $_.Health -match 'unhealthy' }).Count
                restartedCount     = [int]@($g | Where-Object { $_.Restarts -gt 0 }).Count
                cpuPercent         = $cpu
                memoryBytes        = $mem
                writableLayerBytes = $wl
                volumeNames        = $volNames
                volumeBytes        = $volBytes
            }
        }
    )

    $allVolBytes = [int64]0
    $seenVol = @{}
    $uniqueVolNames = [System.Collections.Generic.List[string]]::new()
    foreach ($row in $processedData) {
        foreach ($vn in @($row.VolumeNames)) {
            if (-not $vn -or $seenVol.ContainsKey($vn)) { continue }
            $seenVol[$vn] = $true
            [void]$uniqueVolNames.Add($vn)
            if ($volumeSizeMap.ContainsKey($vn)) { $allVolBytes += [int64]$volumeSizeMap[$vn].Bytes }
        }
    }
    $uniqueVolNames = @($uniqueVolNames | Sort-Object)

    $payload = [ordered]@{
        schemaVersion = 1
        source        = 'powershell'
        capturedAt    = (Get-Date).ToUniversalTime().ToString('o')
        containers    = $containerRows
        stacks        = $stackRows
        totals        = [ordered]@{
            containerCount     = [int]@($processedData).Count
            runningCount       = [int]@($processedData | Where-Object { $_.State -eq 'running' }).Count
            cpuPercent         = [double]($processedData | Where-Object { $_.State -eq 'running' } | Measure-Object CPUVal -Sum).Sum
            memoryBytes        = [int64]($processedData | Where-Object { $_.State -eq 'running' } | Measure-Object MemBytes -Sum).Sum
            writableLayerBytes = [int64]($processedData | Measure-Object DiskBytes -Sum).Sum
            uniqueVolumeNames  = $uniqueVolNames
            uniqueVolumeBytes  = $allVolBytes
        }
    }

    $json = $payload | ConvertTo-Json -Depth 8 -Compress:$false
    $dir = Split-Path -Parent $JsonOut
    if ($dir -and -not (Test-Path -LiteralPath $dir)) {
        New-Item -ItemType Directory -Path $dir | Out-Null
    }
    Set-Content -LiteralPath $JsonOut -Value $json -Encoding UTF8
    Write-Host ("Parity JSON written: {0}" -f $JsonOut) -ForegroundColor Green
    return
}

$grouped = $processedData | Group-Object Stack | Sort-Object Name

foreach ($group in $grouped) {
    Write-Host ("`n=== STACK: {0} ===" -f $group.Name) -ForegroundColor Yellow

    $group.Group |
        Sort-Object @{ Expression = 'MemBytes'; Descending = $true } |
        Select-Object Container, Service, Image, Health, State, Restarts, Uptime, CPU, Memory, Disk |
        Format-Table -AutoSize -Wrap

    $group.Group |
        Sort-Object Container |
        Select-Object Container, External, Internal, IP, Networks, Volumes, NetIO, BlockIO |
        Format-Table -AutoSize -Wrap

    # Итоги стека
    $totalDisk = [int64]($group.Group | Measure-Object DiskBytes -Sum).Sum
    $totalMem  = [int64]($group.Group | Where-Object { $_.State -eq 'running' } | Measure-Object MemBytes -Sum).Sum
    $totalCpu  = [double]($group.Group | Where-Object { $_.State -eq 'running' } | Measure-Object CPUVal -Sum).Sum
    $withVols  = @($group.Group | Where-Object { $_.Volumes -ne '-' }).Count
    $unhealthy = @($group.Group | Where-Object { $_.Health -match 'unhealthy' }).Count
    $restarting = @($group.Group | Where-Object { $_.Restarts -gt 0 }).Count

    Write-Host ("Containers: {0}  |  with volumes: {1}  |  unhealthy: {2}  |  restarts>0: {3}" -f `
        $group.Count, $withVols, $unhealthy, $restarting) -ForegroundColor Gray
    Write-Host ("Stack RAM (running): {0}  |  Stack CPU: {1:N2}%" -f (Format-Bytes $totalMem), $totalCpu) -ForegroundColor Cyan
    Write-Host ("Stack Disk writable: {0}" -f (Format-Bytes $totalDisk)) -ForegroundColor Cyan

    # Топ по RAM
    $top = @($group.Group |
        Where-Object { $_.MemBytes -gt 0 } |
        Sort-Object MemBytes -Descending |
        Select-Object -First 3)
    if ($top.Count) {
        $topText = ($top | ForEach-Object { "{0} ({1})" -f $_.Container, $_.Memory }) -join '  |  '
        Write-Host ("Top RAM: {0}" -f $topText) -ForegroundColor Magenta
    }

    # Тома стека + размеры
    $stackVolNames = @(
        $group.Group |
            ForEach-Object { $_.VolumeNames } |
            Where-Object { $_ } |
            Select-Object -Unique |
            Sort-Object
    )
    if ($stackVolNames.Count) {
        Write-Host 'Volumes (stack):' -ForegroundColor DarkCyan
        $stackVolBytes = [int64]0
        foreach ($vn in $stackVolNames) {
            $meta = $volumeSizeMap[$vn]
            $sizeText = if ($meta) { $meta.Size } else { '?' }
            $bytes = if ($meta) { [int64]$meta.Bytes } else { [int64]0 }
            $stackVolBytes += $bytes
            $links = if ($meta) { $meta.Links } else { '?' }
            Write-Host ("  - {0}  |  {1}  |  links={2}" -f (Format-VolumeName $vn), $sizeText, $links)
        }
        Write-Host ("  = Volume data total: {0}" -f (Format-Bytes $stackVolBytes)) -ForegroundColor Cyan
    } else {
        Write-Host 'Volumes (stack): -' -ForegroundColor DarkCyan
    }

    Write-Host ('-' * 72)
}

# Grand totals
$grandDisk = [int64]($processedData | Measure-Object DiskBytes -Sum).Sum
$grandMem  = [int64]($processedData | Where-Object { $_.State -eq 'running' } | Measure-Object MemBytes -Sum).Sum
$grandCpu  = [double]($processedData | Where-Object { $_.State -eq 'running' } | Measure-Object CPUVal -Sum).Sum
$allVolBytes = [int64]0
$seenVol = @{}
foreach ($row in $processedData) {
    foreach ($vn in @($row.VolumeNames)) {
        if (-not $vn -or $seenVol.ContainsKey($vn)) { continue }
        $seenVol[$vn] = $true
        if ($volumeSizeMap.ContainsKey($vn)) { $allVolBytes += [int64]$volumeSizeMap[$vn].Bytes }
    }
}

Write-Host ''
Write-Host ("ALL containers: {0}" -f @($processedData).Count) -ForegroundColor Green
Write-Host ("ALL RAM (running): {0}  |  ALL CPU: {1:N2}%" -f (Format-Bytes $grandMem), $grandCpu) -ForegroundColor Green
Write-Host ("ALL writable layers: {0}" -f (Format-Bytes $grandDisk)) -ForegroundColor Green
Write-Host ("ALL volume data (unique): {0}" -f (Format-Bytes $allVolBytes)) -ForegroundColor Green
