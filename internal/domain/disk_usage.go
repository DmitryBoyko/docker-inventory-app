package domain

// DiskUsageView is a normalized /system/df summary for the API.
type DiskUsageView struct {
	Volumes    DiskUsageCategory `json:"volumes"`
	Containers DiskUsageCategory `json:"containers"`
	Images     DiskUsageCategory `json:"images"`
	BuildCache DiskUsageCategory `json:"buildCache"`
}

// DiskUsageCategory mirrors Docker df category totals.
type DiskUsageCategory struct {
	ActiveCount  int64 `json:"activeCount"`
	TotalCount   int64 `json:"totalCount"`
	Reclaimable  int64 `json:"reclaimableBytes"`
	TotalSize    int64 `json:"totalSizeBytes"`
	ItemsKnown   bool  `json:"itemsKnown"` // true when per-item usage was available from daemon
}
