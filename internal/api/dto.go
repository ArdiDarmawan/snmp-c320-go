package api

type APIError struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

type OKResponse struct {
	OK bool `json:"ok"`
}

type OltInfo struct {
	Name   string `json:"name"`
	Vendor string `json:"vendor"`
	Model  string `json:"model"`
	Host   string `json:"host"`
	Port   uint16 `json:"port"`
	Version string `json:"version"`
}

type SystemStatus struct {
	SysDescr  any `json:"sys_descr"`
	SysUptime any `json:"sys_uptime"`
	SysName   any `json:"sys_name"`

	CPUUsage    any `json:"cpu_usage"`
	MemoryUsed  any `json:"memory_used"`
	MemoryFree  any `json:"memory_free"`
	Temperature any `json:"temperature"`
}

type PONRef struct {
	Slot        int    `json:"slot"`
	BoardType   string `json:"board_type"`
	PonID       int    `json:"pon_id"`
	Enabled     bool   `json:"enabled"`
	IfIndex1082 uint32 `json:"ifindex_1082"`
	IfIndex1012 uint32 `json:"ifindex_1012"`
}

type ONUBrief struct {
	OnuID   uint32 `json:"onu_id"`
	Name    any    `json:"name"`

	Serial string `json:"serial,omitempty"`

	Status     any    `json:"status,omitempty"`
	StatusText string `json:"status_text,omitempty"`

	RxPowerRaw any    `json:"rx_power_raw,omitempty"`
	RxPowerDbm string `json:"rx_power_dbm,omitempty"`

	LastOfflineReason     any    `json:"last_offline_reason,omitempty"`
	LastOfflineReasonText string `json:"last_offline_reason_text,omitempty"`
}

