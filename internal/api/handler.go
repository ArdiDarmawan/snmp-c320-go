package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gosnmp/gosnmp"

	"zte-c320-snmp-api/internal/cfg"
	"zte-c320-snmp-api/internal/snmp"
)

type Handlers struct {
	Loader *cfg.Loader
}

/* =========================
   RESPONSE WRAPPER (compat frontend)
========================= */

func okResp(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"status": "OK",
		"data":   data,
	})
}

func errResp(c *gin.Context, httpCode int, msg string) {
	c.JSON(httpCode, gin.H{
		"code":   httpCode,
		"status": msg,
		"data":   nil,
	})
}

/* =========================
   CONFIG + OLT SELECTOR
========================= */

func (h *Handlers) conf() *cfg.Config {
	if h.Loader == nil {
		return nil
	}
	return h.Loader.Get()
}

func (h *Handlers) getOltByParam(c *gin.Context) (*cfg.Olt, bool) {
	conf := h.conf()
	if conf == nil {
		return nil, false
	}
	// ✅ IMPORTANT: pakai ResolveOlt (defaults + olt item)
	return conf.ResolveOlt(c.Param("name"))
}

/* =========================
   BASIC
========================= */

func (h *Handlers) Health(c *gin.Context) {
	okResp(c, gin.H{"ok": true})
}

func (h *Handlers) GetOlts(c *gin.Context) {
	conf := h.conf()
	if conf == nil {
		errResp(c, 500, "Config not loaded")
		return
	}

	out := make([]gin.H, 0, len(conf.Olts))
	for _, item := range conf.Olts {
		out = append(out, gin.H{
			"name":    item.Name,
			"vendor":  conf.Defaults.Vendor, // ✅ dari defaults
			"model":   conf.Defaults.Model,  // ✅ dari defaults
			"host":    item.SNMP.Host,
			"port":    item.SNMP.Port,
			"version": item.SNMP.Version,
		})
	}
	okResp(c, out)
}

/* =========================
   SYSTEM (basic from config.yaml)
========================= */

func (h *Handlers) GetSystem(c *gin.Context) {
	olt, ok := h.getOltByParam(c)
	if !ok {
		errResp(c, 404, "OLT not found")
		return
	}
	o := *olt

	cl, err := snmp.New(o.SNMP)
	if err != nil {
		errResp(c, 502, err.Error())
		return
	}
	defer cl.Close()

	// standard MIB-2 (FULL)
	sysDescr := o.SystemOIDs.SysDescr
	sysUp := o.SystemOIDs.SysUptime
	sysName := o.SystemOIDs.SysName

	// vendor relative (join base)
	cpuOID := snmp.JoinBaseRel(o.BaseOIDs.OID1015, o.SystemOIDs.CPUUsage)
	memUsedOID := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.SystemOIDs.MemoryUsed)
	memFreeOID := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.SystemOIDs.MemoryFree)

	// ✅ FIX: Temp -> Temperature
	tempOID := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.SystemOIDs.Temperature)

	got, err := cl.Get(sysDescr, sysUp, sysName, cpuOID, memUsedOID, memFreeOID, tempOID)
	if err != nil {
		errResp(c, 502, err.Error())
		return
	}

	okResp(c, gin.H{
		"sys_descr":   toString(pduValue(got[sysDescr])),
		"sys_uptime":  pduValue(got[sysUp]),
		"sys_name":    toString(pduValue(got[sysName])),
		"cpu_usage":   pduValue(got[cpuOID]),
		"memory_used": pduValue(got[memUsedOID]),
		"memory_free": pduValue(got[memFreeOID]),
		"temperature": pduValue(got[tempOID]),
	})
}

/* =========================
   SYSTEM HEALTH (full OIDs + uptime human)
   GET /.../system/health
========================= */

func (h *Handlers) GetSystemHealth(c *gin.Context) {
	olt, ok := h.getOltByParam(c)
	if !ok {
		errResp(c, 404, "OLT not found")
		return
	}
	o := *olt

	cl, err := snmp.New(o.SNMP)
	if err != nil {
		errResp(c, 502, err.Error())
		return
	}
	defer cl.Close()

	// ===== OIDs (FULL) - sesuai list kamu
	oidSysDescr := "1.3.6.1.2.1.1.1.0"
	oidSysUptime := "1.3.6.1.2.1.1.3.0"
	oidSysName := "1.3.6.1.2.1.1.5.0"

	oidSoftwareVersionBoard1 := "1.3.6.1.4.1.3902.1015.2.1.2.2.1.4.1.1"
	oidOltSerial := "1.3.6.1.4.1.3902.1015.2.1.1.2.1.5.1.1"

	oidTempC := "1.3.6.1.4.1.3902.1015.2.1.3.2.0"
	oidFanRpmBase := "1.3.6.1.4.1.3902.1015.2.1.3.10.10.10.1.7"

	oidDcVoltBase := "1.3.6.1.4.1.3902.1082.10.10.2.1.6.1.4.1.1"
	oidPsuWattBase := "1.3.6.1.4.1.3902.1082.10.10.2.1.6.1.3.1.1"
	oidPsuAmpBase := "1.3.6.1.4.1.3902.1082.10.10.2.1.6.1.5.1.1"

	// GET singles
	pdus, err := cl.Get(
		oidSysDescr, oidSysUptime, oidSysName,
		oidSoftwareVersionBoard1, oidOltSerial,
		oidTempC,
	)
	if err != nil {
		errResp(c, 502, err.Error())
		return
	}

	// robust lookup by normalized PDU name
	pduByName := map[string]gosnmp.SnmpPDU{}
	for _, p := range pdus {
		pduByName[normOID(p.Name)] = p
	}
	getVal := func(oid string) any {
		if p, ok := pduByName[normOID(oid)]; ok {
			return pduValue(p)
		}
		return nil
	}

	// TimeTicks -> human
	timeticksToDuration := func(v any) (time.Duration, bool) {
		n, ok := toInt64(v)
		if !ok || n <= 0 {
			return 0, false
		}
		// TimeTicks = 1/100 detik
		return time.Duration(n) * 10 * time.Millisecond, true
	}

	uptimeRaw := getVal(oidSysUptime)
	uptime := gin.H{
		"ticks":   uptimeRaw,
		"seconds": nil,
		"human":   "",
	}
	if d, ok := timeticksToDuration(uptimeRaw); ok {
		uptime["seconds"] = int64(d.Seconds())
		uptime["human"] = formatDurationDHMS(d)
	}

	// WALK tables
	fansPdus, _ := cl.WalkAll(oidFanRpmBase)
	dcPdus, _ := cl.WalkAll(oidDcVoltBase)
	wattPdus, _ := cl.WalkAll(oidPsuWattBase)
	ampPdus, _ := cl.WalkAll(oidPsuAmpBase)

	// fans
	fans := []gin.H{}
	for _, pdu := range fansPdus {
		idx := oidLastIndex(pdu.Name)
		rpm, ok := toInt64(pduValue(pdu))
		if !ok {
			continue
		}
		fans = append(fans, gin.H{"fan": idx, "rpm": rpm})
	}

	// scaled table raw * 0.001 ; skip raw==0
	buildScaledTable := func(pdus []gosnmp.SnmpPDU) []gin.H {
		out := []gin.H{}
		for _, pdu := range pdus {
			idx := oidLastIndex(pdu.Name)
			raw, ok := toInt64(pduValue(pdu))
			if !ok || raw == 0 {
				continue
			}
			out = append(out, gin.H{
				"channel": idx,
				"raw":     raw,
				"value":   float64(raw) * 0.001,
			})
		}
		return out
	}

	dcTable := buildScaledTable(dcPdus)
	wattTable := buildScaledTable(wattPdus)

	// current: -1000 => N/A ; 0 => inactive skip
	ampTable := []gin.H{}
	for _, pdu := range ampPdus {
		idx := oidLastIndex(pdu.Name)
		raw, ok := toInt64(pduValue(pdu))
		if !ok || raw == 0 {
			continue
		}
		if raw == -1000 {
			ampTable = append(ampTable, gin.H{
				"channel": idx,
				"raw":     raw,
				"value":   nil,
				"note":    "N/A",
			})
			continue
		}
		ampTable = append(ampTable, gin.H{
			"channel": idx,
			"raw":     raw,
			"value":   float64(raw) * 0.001,
		})
	}

	okResp(c, gin.H{
		"identity": gin.H{
			"sys_descr":  toString(getVal(oidSysDescr)),
			"sys_name":   toString(getVal(oidSysName)),
			"sys_uptime": uptime,
		},
		"software": gin.H{
			"olt_serial_number":       toString(getVal(oidOltSerial)),
			"software_version_board1": toString(getVal(oidSoftwareVersionBoard1)),
		},
		"environment": gin.H{
			"temperature_c": getVal(oidTempC),
		},
		"fans": fans,
		"power": gin.H{
			"dc_voltage":      dcTable,
			"psu_output_watt": wattTable,
			"psu_current":     ampTable,
		},
	})
}

/* =========================
   PON LIST (from config)
========================= */

func (h *Handlers) ListPons(c *gin.Context) {
	olt, ok := h.getOltByParam(c)
	if !ok {
		errResp(c, 404, "OLT not found")
		return
	}
	o := *olt

	var out []gin.H
	for _, b := range o.Boards {
		for _, p := range b.Pons {
			out = append(out, gin.H{
				"slot":         b.Slot,
				"board_type":   b.Type,
				"pon_id":       p.PonID,
				"enabled":      p.Enabled,
				"ifindex_1082": p.IfIndex1082,
				"ifindex_1012": p.IfIndex1012,
			})
		}
	}
	okResp(c, out)
}

/* =========================
   ONU LIST (by slot/pon from config)
========================= */

func (h *Handlers) ListOnusByPon(c *gin.Context) {
	olt, ok := h.getOltByParam(c)
	if !ok {
		errResp(c, 404, "OLT not found")
		return
	}
	o := *olt

	slot, err := strconv.Atoi(c.Param("slot"))
	if err != nil {
		errResp(c, 400, "slot must be integer")
		return
	}
	ponID, err := strconv.Atoi(c.Param("pon"))
	if err != nil {
		errResp(c, 400, "pon must be integer")
		return
	}

	detail := c.Query("detail") == "1" || strings.EqualFold(c.Query("detail"), "true")

	pon, ok := findPon(o.Boards, slot, ponID)
	if !ok || !pon.Enabled {
		okResp(c, []gin.H{})
		return
	}

	cl, err := snmp.New(o.SNMP)
	if err != nil {
		errResp(c, 502, err.Error())
		return
	}
	defer cl.Close()

	// WALK ONU name table: oid_1082 + onu_id_name + .<ifIndex1082>
	nameBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuIDName)

	// ✅ FIX: JoinIndexes expects uint32
	walkBase := snmp.JoinIndexes(nameBase, uint32(pon.IfIndex1082))

	pdus, err := cl.WalkAll(walkBase)
	if err != nil {
		errResp(c, 502, err.Error())
		return
	}

	list := make([]gin.H, 0, len(pdus))
	for _, pdu := range pdus {
		onuNo, ok := extractOnuIDFromOID(pdu.Name)
		if !ok {
			continue
		}
		name := strings.TrimSpace(toString(pduValue(pdu)))
		if name == "" {
			name = fmt.Sprintf("ONU-%d:%d", ponID, onuNo)
		}

		list = append(list, gin.H{
			"board":  slot,
			"pon":    ponID,
			"onu_id": onuNo, // numeric default (uint32)
			"name":   name,
		})
	}

	// kalau tidak detail, langsung balikin
	if !detail || len(list) == 0 {
		okResp(c, list)
		return
	}

	// detail fields (per ONU)
	serialBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuSerialNumber)
	statusBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuStatusID)
	rxBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuRxPower)

	typeBase := snmp.JoinBaseRel(o.BaseOIDs.OID1012, o.OnuOIDs.OID1012.OnuType)

	// tx power source = OLT RX power (1015)
	oltRxBase := snmp.JoinBaseRel(o.BaseOIDs.OID1015, o.SystemOIDs.OltRxPowerBase)

	for i := range list {
		onuNo := list[i]["onu_id"].(uint32)

		oidType := snmp.JoinIndexes(typeBase, uint32(pon.IfIndex1012), onuNo)
		oidSerial := snmp.JoinIndexes(serialBase, uint32(pon.IfIndex1082), onuNo)
		oidStatus := snmp.JoinIndexes(statusBase, uint32(pon.IfIndex1082), onuNo)
		oidRx := snmp.JoinIndexes(rxBase, uint32(pon.IfIndex1082), onuNo, 1)
		oidTx := snmp.JoinIndexes(oltRxBase, uint32(pon.IfIndex1012), onuNo)

		got, err := cl.Get(oidType, oidSerial, oidStatus, oidRx, oidTx)
		if err != nil {
			continue
		}

		if pdu, ok := got[oidType]; ok {
			list[i]["onu_type"] = strings.TrimSpace(toString(pduValue(pdu)))
		}
		if pdu, ok := got[oidSerial]; ok {
			list[i]["serial_number"] = strings.TrimSpace(normalizeSerial(pduValue(pdu)))
		}
		if pdu, ok := got[oidStatus]; ok {
			list[i]["status"] = statusTextFromCode(pduValue(pdu))
		}
		if pdu, ok := got[oidRx]; ok {
			if s, err := convertRxDbm(pduValue(pdu)); err == nil {
				list[i]["rx_power"] = s
			}
		}
		if pdu, ok := got[oidTx]; ok {
			if dbm, ok := convertDbmFromScaledInt(pduValue(pdu)); ok {
				// tx_power as number (float)
				list[i]["tx_power"] = float64(int(dbm*100)) / 100
			}
		}

		// ubah onu_id jadi string format "ONU<pon>:<onu>"
		list[i]["onu_id"] = fmt.Sprintf("ONU%d:%d", ponID, onuNo)
	}

	okResp(c, list)
}

/* =========================
   ONU DETAIL
========================= */

func (h *Handlers) GetOnuDetail(c *gin.Context) {
	olt, ok := h.getOltByParam(c)
	if !ok {
		errResp(c, 404, "OLT not found")
		return
	}
	o := *olt

	slot, err := strconv.Atoi(c.Param("slot"))
	if err != nil {
		errResp(c, 400, "slot must be integer")
		return
	}
	ponID, err := strconv.Atoi(c.Param("pon"))
	if err != nil {
		errResp(c, 400, "pon must be integer")
		return
	}
	onuID64, err := strconv.ParseUint(c.Param("onuId"), 10, 32)
	if err != nil {
		errResp(c, 400, "onuId must be integer")
		return
	}
	onuID := uint32(onuID64)

	pon, ok := findPon(o.Boards, slot, ponID)
	if !ok || !pon.Enabled {
		errResp(c, 404, "PON not found or disabled")
		return
	}

	cl, err := snmp.New(o.SNMP)
	if err != nil {
		errResp(c, 502, err.Error())
		return
	}
	defer cl.Close()

	// 1082
	nameBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuIDName)
	serialBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuSerialNumber)
	statusBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuStatusID)
	descBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuDescription)
	lastOnBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuLastOnlineTime)
	lastOffBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuLastOfflineTime)
	reasonBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuLastOfflineReason)
	distBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuOpticalDistance)
	rxBase := snmp.JoinBaseRel(o.BaseOIDs.OID1082, o.OnuOIDs.OID1082.OnuRxPower)

	oidName := snmp.JoinIndexes(nameBase, uint32(pon.IfIndex1082), onuID)
	oidSerial := snmp.JoinIndexes(serialBase, uint32(pon.IfIndex1082), onuID)
	oidStatus := snmp.JoinIndexes(statusBase, uint32(pon.IfIndex1082), onuID)
	oidDesc := snmp.JoinIndexes(descBase, uint32(pon.IfIndex1082), onuID)
	oidLastOn := snmp.JoinIndexes(lastOnBase, uint32(pon.IfIndex1082), onuID)
	oidLastOff := snmp.JoinIndexes(lastOffBase, uint32(pon.IfIndex1082), onuID)
	oidReason := snmp.JoinIndexes(reasonBase, uint32(pon.IfIndex1082), onuID)
	oidDist := snmp.JoinIndexes(distBase, uint32(pon.IfIndex1082), onuID)
	oidRx := snmp.JoinIndexes(rxBase, uint32(pon.IfIndex1082), onuID, 1)

	// 1012
	typeBase := snmp.JoinBaseRel(o.BaseOIDs.OID1012, o.OnuOIDs.OID1012.OnuType)
	ipBase := snmp.JoinBaseRel(o.BaseOIDs.OID1012, o.OnuOIDs.OID1012.OnuIPAddress)

	oidType := snmp.JoinIndexes(typeBase, uint32(pon.IfIndex1012), onuID)
	oidIP := snmp.JoinIndexes(ipBase, uint32(pon.IfIndex1012), onuID)

	// 1015 (tx power source = OLT RX power)
	oltRxBase := snmp.JoinBaseRel(o.BaseOIDs.OID1015, o.SystemOIDs.OltRxPowerBase)
	oidTx := snmp.JoinIndexes(oltRxBase, uint32(pon.IfIndex1012), onuID)

	got, err := cl.Get(
		oidName, oidDesc, oidType, oidSerial,
		oidStatus, oidIP,
		oidRx, oidTx,
		oidLastOn, oidLastOff, oidReason, oidDist,
	)
	if err != nil {
		errResp(c, 502, err.Error())
		return
	}

	get := func(oid string) (any, bool) {
		pdu, ok := got[oid]
		if !ok {
			return nil, false
		}
		return pduValue(pdu), true
	}

	data := gin.H{
		"board":  slot,
		"pon":    ponID,
		"onu_id": onuID,
	}

	// name + description
	name := fmt.Sprintf("ONU-%d:%d", ponID, onuID)
	if v, ok := get(oidName); ok {
		s := strings.TrimSpace(toString(v))
		if s != "" {
			name = s
		}
	}
	data["name"] = name

	desc := name
	if v, ok := get(oidDesc); ok {
		s := strings.TrimSpace(toString(v))
		if s != "" {
			desc = s
		}
	}
	data["description"] = desc

	// onu_type
	if v, ok := get(oidType); ok {
		s := strings.TrimSpace(toString(v))
		if s != "" {
			data["onu_type"] = s
		}
	}

	// serial
	if v, ok := get(oidSerial); ok {
		s := strings.TrimSpace(normalizeSerial(v))
		if s != "" {
			data["serial_number"] = s
		}
	}

	// rx_power
	if v, ok := get(oidRx); ok {
		if s, err := convertRxDbm(v); err == nil {
			data["rx_power"] = s
		}
	}

	// tx_power (from OLT RX power)
	if v, ok := get(oidTx); ok {
		if dbm, ok := convertDbmFromScaledInt(v); ok {
			data["tx_power"] = fmt.Sprintf("%.2f", dbm)
		}
	}

	// status
	if v, ok := get(oidStatus); ok {
		data["status"] = statusTextFromCode(v)
	}

	// ip
	if v, ok := get(oidIP); ok {
		s := strings.TrimSpace(toString(v))
		if s != "" {
			data["ip_address"] = s
		}
	}

	// last online/offline + durations
	var lastOn time.Time
	var lastOff time.Time

	if v, ok := get(oidLastOn); ok {
		if b, ok := v.([]byte); ok {
			if t, ok := decodeDateAndTimeBytes(b); ok {
				lastOn = t
				data["last_online"] = t.Format("2006-01-02 15:04:05")
			}
		}
	}
	if v, ok := get(oidLastOff); ok {
		if b, ok := v.([]byte); ok {
			if t, ok := decodeDateAndTimeBytes(b); ok {
				lastOff = t
				data["last_offline"] = t.Format("2006-01-02 15:04:05")
			}
		}
	}

	if !lastOn.IsZero() {
		data["uptime"] = formatDurationDHMS(time.Since(lastOn))
	}
	if !lastOn.IsZero() && !lastOff.IsZero() {
		data["last_down_time_duration"] = formatDurationDHMS(lastOn.Sub(lastOff))
	}

	// offline reason
	if v, ok := get(oidReason); ok {
		data["offline_reason"] = offlineReasonTextFromCode(v)
	}

	// distance
	if v, ok := get(oidDist); ok {
		data["gpon_optical_distance"] = fmt.Sprint(v)
	}

	okResp(c, data)
}

/* =========================
   HELPERS
========================= */

func findPon(boards []cfg.Board, slot int, ponID int) (cfg.PON, bool) {
	for _, b := range boards {
		if b.Slot != slot {
			continue
		}
		for _, p := range b.Pons {
			if p.PonID == ponID {
				return p, true
			}
		}
	}
	return cfg.PON{}, false
}

func pduValue(pdu gosnmp.SnmpPDU) any {
	return pdu.Value
}
