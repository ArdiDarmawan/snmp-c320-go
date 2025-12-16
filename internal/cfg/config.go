package cfg

import "strings"

/*
====================================================
ROOT CONFIG
====================================================
*/

type Config struct {
	Defaults OltDefaults `yaml:"defaults"`
	Olts     []OltItem   `yaml:"olts"`
}

/*
====================================================
OLT DEFAULTS (STATIC)
====================================================
*/

type OltDefaults struct {
	Vendor     string     `yaml:"vendor"`
	Model      string     `yaml:"model"`
	BaseOIDs   BaseOIDs   `yaml:"base_oids"`
	SystemOIDs SystemOIDs `yaml:"system_oids"`
	OnuOIDs    OnuOIDs    `yaml:"onu_oids"`
	Boards     []Board    `yaml:"boards"`
}

/*
====================================================
OLT LIST ITEM (DYNAMIC)
====================================================
*/

type OltItem struct {
	Name string     `yaml:"name"`
	SNMP SNMPConfig `yaml:"snmp"`
}

/*
====================================================
OLT RUNTIME (RESOLVED)
====================================================
*/

type Olt struct {
	Name       string
	Vendor     string
	Model      string
	SNMP       SNMPConfig
	BaseOIDs   BaseOIDs
	SystemOIDs SystemOIDs
	OnuOIDs    OnuOIDs
	Boards     []Board
}

/*
====================================================
SNMP CONFIG (NAMA DISESUAIKAN: SNMPConfig)
====================================================
*/

type SNMPConfig struct {
	Version   string `yaml:"version"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Community string `yaml:"community"`
	TimeoutMS int    `yaml:"timeout_ms"`
	Retries   int    `yaml:"retries"`
}

/*
====================================================
BASE OIDS
====================================================
*/

type BaseOIDs struct {
	OID1082 string `yaml:"oid_1082"`
	OID1012 string `yaml:"oid_1012"`
	OID1015 string `yaml:"oid_1015"`
}

/*
====================================================
SYSTEM OIDS
====================================================
*/

type SystemOIDs struct {
	SysDescr       string `yaml:"sys_descr"`
	SysUptime      string `yaml:"sys_uptime"`
	SysName        string `yaml:"sys_name"`
	CPUUsage       string `yaml:"cpu_usage"`
	MemoryUsed     string `yaml:"memory_used"`
	MemoryFree     string `yaml:"memory_free"`
	Temperature    string `yaml:"temperature"`
	OltRxPowerBase string `yaml:"olt_rx_power_base"`
}

/*
====================================================
ONU OIDS
====================================================
*/

type OnuOIDs struct {
	OID1082 OnuOID1082 `yaml:"oid_1082"`
	OID1012 OnuOID1012 `yaml:"oid_1012"`
}

type OnuOID1082 struct {
	OnuIDName            string `yaml:"onu_id_name"`
	OnuSerialNumber      string `yaml:"onu_serial_number"`
	OnuRxPower           string `yaml:"onu_rx_power"`
	OnuStatusID          string `yaml:"onu_status_id"`
	OnuDescription       string `yaml:"onu_description"`
	OnuLastOnlineTime    string `yaml:"onu_last_online_time"`
	OnuLastOfflineTime   string `yaml:"onu_last_offline_time"`
	OnuLastOfflineReason string `yaml:"onu_last_offline_reason"`
	OnuOpticalDistance   string `yaml:"onu_optical_distance"`
}

type OnuOID1012 struct {
	OnuType      string `yaml:"onu_type"`
	OnuTxPower   string `yaml:"onu_tx_power"`
	OnuIPAddress string `yaml:"onu_ip_address"`
}

/*
====================================================
BOARD & PON
====================================================
*/

type Board struct {
	Slot int   `yaml:"slot"`
	Type string `yaml:"type"`
	Pons []PON `yaml:"pons"`
}

type PON struct {
	PonID       int  `yaml:"pon_id"`
	Enabled     bool `yaml:"enabled"`
	IfIndex1082 int  `yaml:"ifindex_1082"`
	IfIndex1012 int  `yaml:"ifindex_1012"`
}

/*
====================================================
FIND & RESOLVE OLT
====================================================
*/

// Cari OLT dari list (case-insensitive)
func (c *Config) FindOltByName(name string) (*OltItem, bool) {
	if c == nil {
		return nil, false
	}
	for i := range c.Olts {
		if strings.EqualFold(c.Olts[i].Name, name) {
			return &c.Olts[i], true
		}
	}
	return nil, false
}

// Gabungkan defaults + OLT item → OLT runtime siap dipakai
func (c *Config) ResolveOlt(name string) (*Olt, bool) {
	item, ok := c.FindOltByName(name)
	if !ok {
		return nil, false
	}

	return &Olt{
		Name:       item.Name,
		Vendor:     c.Defaults.Vendor,
		Model:      c.Defaults.Model,
		SNMP:       item.SNMP,
		BaseOIDs:   c.Defaults.BaseOIDs,
		SystemOIDs: c.Defaults.SystemOIDs,
		OnuOIDs:    c.Defaults.OnuOIDs,
		Boards:     c.Defaults.Boards,
	}, true
}
