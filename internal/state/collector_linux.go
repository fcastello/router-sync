//go:build linux

package state

import (
	"fmt"
	"sort"
	"strconv"

	"router-sync/internal/models"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// Collect builds a complete RouterState snapshot on Linux using netlink.
func (c *Collector) Collect() (*models.RouterState, error) {
	state := &models.RouterState{
		Hostname: c.hostname,
	}

	if ifaces, err := c.collectInterfaces(); err != nil {
		logrus.Warnf("Failed to collect interfaces: %v", err)
	} else {
		state.Interfaces = ifaces
	}

	if tables, err := c.collectTables(); err != nil {
		logrus.Warnf("Failed to collect routing tables: %v", err)
	} else {
		state.Tables = tables
	}

	if rules, err := c.collectRules(); err != nil {
		logrus.Warnf("Failed to collect ip rules: %v", err)
	} else {
		state.Rules = rules
	}

	return state, nil
}

func (c *Collector) collectInterfaces() ([]models.Interface, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("failed to list links: %w", err)
	}

	out := make([]models.Interface, 0, len(links))
	for _, link := range links {
		attrs := link.Attrs()
		if attrs == nil {
			continue
		}

		iface := models.Interface{
			Name: attrs.Name,
			MTU:  attrs.MTU,
			Up:   attrs.Flags&unix.IFF_UP != 0,
			MAC:  attrs.HardwareAddr.String(),
		}

		addrs, err := netlink.AddrList(link, unix.AF_UNSPEC)
		if err == nil {
			for _, a := range addrs {
				if a.IPNet != nil {
					iface.Addresses = append(iface.Addresses, a.IPNet.String())
				}
			}
		}
		out = append(out, iface)
	}
	return out, nil
}

func (c *Collector) collectTables() ([]models.RoutingTable, error) {
	// netlink.RouteList only returns the main table (it skips every route whose
	// table != RT_TABLE_MAIN unless RT_FILTER_TABLE is set). Pass an unspecified
	// table with RT_FILTER_TABLE so the kernel dumps every table and the library
	// keeps them all — this is what surfaces the per-provider tables (99/100/200).
	routes, err := netlink.RouteListFiltered(
		unix.AF_UNSPEC,
		&netlink.Route{Table: unix.RT_TABLE_UNSPEC},
		netlink.RT_FILTER_TABLE,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}

	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("failed to list links for route mapping: %w", err)
	}
	linkByIndex := make(map[int]string, len(links))
	for _, link := range links {
		if link.Attrs() != nil {
			linkByIndex[link.Attrs().Index] = link.Attrs().Name
		}
	}

	byTable := make(map[int]*models.RoutingTable)
	for _, r := range routes {
		tbl, ok := byTable[r.Table]
		if !ok {
			tbl = &models.RoutingTable{
				ID:   r.Table,
				Name: c.tableNames[r.Table],
			}
			byTable[r.Table] = tbl
		}
		tbl.Routes = append(tbl.Routes, routeToModel(r, linkByIndex))
	}

	out := make([]models.RoutingTable, 0, len(byTable))
	for _, t := range byTable {
		out = append(out, *t)
	}
	// Stable ordering so the UI doesn't reshuffle tables between heartbeats:
	// main (254) first, then provider tables in ascending ID order.
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID == unix.RT_TABLE_MAIN {
			return true
		}
		if out[j].ID == unix.RT_TABLE_MAIN {
			return false
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func routeToModel(r netlink.Route, linkByIndex map[int]string) models.Route {
	dst := "default"
	if r.Dst != nil {
		dst = r.Dst.String()
	}
	gw := ""
	if r.Gw != nil {
		gw = r.Gw.String()
	}
	return models.Route{
		Dst:       dst,
		Gateway:   gw,
		Interface: linkByIndex[r.LinkIndex],
		Protocol:  routeProtoString(int(r.Protocol)),
		Scope:     routeScopeString(int(r.Scope)),
		Metric:    r.Priority,
	}
}

func routeProtoString(p int) string {
	switch p {
	case unix.RTPROT_KERNEL:
		return "kernel"
	case unix.RTPROT_BOOT:
		return "boot"
	case unix.RTPROT_STATIC:
		return "static"
	case unix.RTPROT_DHCP:
		return "dhcp"
	default:
		if p == 0 {
			return ""
		}
		return strconv.Itoa(p)
	}
}

func routeScopeString(s int) string {
	switch s {
	case unix.RT_SCOPE_UNIVERSE:
		return "global"
	case unix.RT_SCOPE_SITE:
		return "site"
	case unix.RT_SCOPE_LINK:
		return "link"
	case unix.RT_SCOPE_HOST:
		return "host"
	case unix.RT_SCOPE_NOWHERE:
		return ""
	default:
		return ""
	}
}
