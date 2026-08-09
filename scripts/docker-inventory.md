```PowerShell
powershell -NoProfile -ExecutionPolicy Bypass -File "E:\ProxyServiceYandexLocatorDeploy\scripts\docker-stack-inventory.ps1"
```



На Linux:

```bash
chmod +x docker-stack-inventory.sh
./docker-stack-inventory.sh
```

Нужны: `docker`, `jq`, `column` (обычно уже есть). Если нет jq:

```bash
# Debian/Ubuntu
sudo apt install jq
# RHEL/Fedora
sudo dnf install jq
```

Поведение как у PowerShell-версии: спиннер на время сбора, две таблицы на стек, RAM/CPU/тома/итоги. Текст скрипта в вывод не идёт.

В **Git Bash**:

```bash
cd /e/ProxyServiceYandexLocatorDeploy/scripts
chmod +x docker-stack-inventory.sh
./docker-stack-inventory.sh
```

Или одной строкой:

```bash
bash /e/ProxyServiceYandexLocatorDeploy/scripts/docker-stack-inventory.sh
```

`E:\...` → в Git Bash это `/e/...`.

Если напишет `нужен jq` — в Git Bash его обычно нет. Поставьте, например:

```bash
# через scoop (если есть)
scoop install jq

# или скачайте jq.exe в папку из PATH, например:
# https://github.com/jqlang/jq/releases → jq-windows-amd64.exe
# переименуйте в jq.exe и положите рядом со скриптом / в Git\usr\bin
```

Проверка:

```bash
command -v docker && command -v jq && echo OK
```

cd /e/ProxyServiceYandexLocatorDeploy/scripts
./docker-stack-inventory.sh