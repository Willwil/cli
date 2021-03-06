package cmd

import (
	"github.com/rancher/go-rancher/client"
	"github.com/urfave/cli"
)

func HostCommand() cli.Command {
	return cli.Command{
		Name:      "hosts",
		ShortName: "host",
		Usage:     "Operations on hosts",
		Action:    defaultAction,
		Subcommands: []cli.Command{
			cli.Command{
				Name:            "create",
				Usage:           "Create a host",
				SkipFlagParsing: true,
				Action:          hostCreate,
			},
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "quiet,q",
				Usage: "Only display IDs",
			},
			cli.StringFlag{
				Name:  "format",
				Usage: "'json' or Custom format: {{.Id}} {{.Name}",
			},
		},
	}
}

type HostsData struct {
	ID          string
	Host        client.Host
	State       string
	IPAddresses []client.IpAddress
}

func getHostState(host *client.Host) string {
	state := host.State
	if state == "active" && host.AgentState != "" {
		state = host.AgentState
	}
	return state
}

func defaultAction(ctx *cli.Context) error {
	if ctx.Bool("help") || len(ctx.Args()) > 0 {
		cli.ShowAppHelp(ctx)
		return nil
	}

	return hostLs(ctx)
}

func hostLs(ctx *cli.Context) error {
	c, err := GetClient(ctx)
	if err != nil {
		return err
	}

	collection, err := c.Host.List(nil)
	if err != nil {
		return err
	}

	knownMachines := map[string]bool{}
	for _, host := range collection.Data {
		knownMachines[host.PhysicalHostId] = true
	}

	machineCollection, err := c.Machine.List(nil)
	if err != nil {
		return err
	}

	for _, machine := range machineCollection.Data {
		if knownMachines[machine.Id] {
			continue
		}
		host := client.Host{
			Resource: client.Resource{
				Id: machine.Id,
			},
			Hostname:             machine.Name,
			State:                machine.State,
			TransitioningMessage: machine.TransitioningMessage,
		}
		if machine.State == "active" {
			host.State = "waiting"
			host.TransitioningMessage = "Almost there... Waiting for agent connection"
		}
		collection.Data = append(collection.Data, host)
	}

	writer := NewTableWriter([][]string{
		{"ID", "Host.Id"},
		{"HOSTNAME", "Host.Hostname"},
		{"STATE", "State"},
		{"IP", "{{ips .IPAddresses}}"},
		{"DETAIL", "Host.TransitioningMessage"},
	}, ctx)

	defer writer.Close()

	for _, item := range collection.Data {
		ips := client.IpAddressCollection{}
		// ignore errors getting IPs, machines don't have them
		c.GetLink(item.Resource, "ipAddresses", &ips)

		writer.Write(&HostsData{
			ID:          item.Id,
			Host:        item,
			State:       getHostState(&item),
			IPAddresses: ips.Data,
		})
	}

	return writer.Err()
}
