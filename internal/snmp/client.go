package snmp

import (
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
	"zte-c320-snmp-api/internal/cfg"
)

type Client struct {
	g *gosnmp.GoSNMP
}

func New(c cfg.SNMPConfig) (*Client, error) {
	if c.Version != "2c" && c.Version != "2" && c.Version != "v2c" {
		return nil, fmt.Errorf("only SNMP v2c supported in this template (got %q)", c.Version)
	}

	g := &gosnmp.GoSNMP{
		Target:    c.Host,
		Port:      uint16(c.Port),
		Community: c.Community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(c.TimeoutMS) * time.Millisecond,
		Retries:   c.Retries,
		MaxOids:   gosnmp.MaxOids,
	}
	if err := g.Connect(); err != nil {
		return nil, err
	}
	return &Client{g: g}, nil
}

func (c *Client) Close() error {
	return c.g.Conn.Close()
}

func (c *Client) Get(oids ...string) (map[string]gosnmp.SnmpPDU, error) {
	pkt, err := c.g.Get(oids)
	if err != nil {
		return nil, err
	}
	out := map[string]gosnmp.SnmpPDU{}
	for _, v := range pkt.Variables {
		out[v.Name] = v
	}
	return out, nil
}

// WalkAll returns all PDUs under baseOid
func (c *Client) WalkAll(baseOid string) ([]gosnmp.SnmpPDU, error) {
	var out []gosnmp.SnmpPDU
	err := c.g.Walk(baseOid, func(pdu gosnmp.SnmpPDU) error {
		out = append(out, pdu)
		return nil
	})
	return out, err
}
