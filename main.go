package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"os"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/shirou/gopsutil/v3/disk"
)

type DiskInfo struct {
	Device     string  `json:"device"`
	Mountpoint string  `json:"mountpoint"`
	TotalGB    float64 `json:"total_gb"`
	FreeGB     float64 `json:"free_gb"`
	FreePct    float64 `json:"free_pct"`
}

func main() {
	format := flag.String("format", "text", "Output format: text, json, html")
	flag.Parse()

	partitions, err := disk.Partitions(false)
	if err != nil {
		fmt.Println("Error getting partitions:", err)
		return
	}

	var disks []DiskInfo
	seen := map[string]bool{}

	for _, p := range partitions {
		diskID := normalizeDeviceID(p.Device)

		if seen[diskID] {
			continue // Skip already processed partitions
		}
		seen[diskID] = true

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue // skip unmountable or inaccessible partitions
		}

		info := DiskInfo{
			Device:     diskID,
			Mountpoint: p.Mountpoint,
			TotalGB:    float64(usage.Total) / 1e9,
			FreeGB:     float64(usage.Free) / 1e9,
			FreePct:    float64(usage.Free) / float64(usage.Total) * 100,
		}

		disks = append(disks, info)
	}

	switch *format {
	case "json":
		outputJSON(disks)
	case "html":
		outputHTML(disks)
	case "text":
		fallthrough
	default:
		outputText(disks)
	}
}

func outputText(disks []DiskInfo) {
	title := color.New(color.FgCyan, color.Bold).SprintFunc()
	header := color.New(color.FgGreen, color.Bold).SprintFunc()
	printInfo := color.New(color.FgWhite).SprintFunc()

	fmt.Println(title("\n📦 Disk Usage Summary\n"))

	for _, info := range disks {
		fmt.Printf("%s\n", header(fmt.Sprintf("🔹 Device: %s", info.Device)))
		fmt.Printf("   Mountpoint: %s\n", printInfo(info.Mountpoint))
		fmt.Printf("   Total:      %s\n", header(fmt.Sprintf("%.2f GB", info.TotalGB)))
		fmt.Printf("   Free:       %s\n", getFreeColor(&disk.UsageStat{Free: uint64(info.FreeGB * 1e9), Total: uint64(info.TotalGB * 1e9)})(fmt.Sprintf("%.2f GB (%.2f%%)", info.FreeGB, info.FreePct)))
		fmt.Println()
	}
}

func outputJSON(disks []DiskInfo) {
	data, err := json.MarshalIndent(disks, "", "  ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}
	fmt.Println(string(data))
}

func outputHTML(disks []DiskInfo) {
	const tpl = `
<!DOCTYPE html>
<html>
<head>
	<title>Disk Usage</title>
	<style>
		table { border-collapse: collapse; width: 60%; }
		th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
		th { background-color: #f2f2f2; }
	</style>
</head>
<body>
	<h2>Disk Usage Summary</h2>
	<table>
		<tr>
			<th>Device</th>
			<th>Mountpoint</th>
			<th>Total (GB)</th>
			<th>Free (GB)</th>
			<th>Free (%)</th>
		</tr>
		{{range .}}
		<tr>
			<td>{{.Device}}</td>
			<td>{{.Mountpoint}}</td>
			<td>{{printf "%.2f" .TotalGB}}</td>
			<td>{{printf "%.2f" .FreeGB}}</td>
			<td>{{printf "%.0f" .FreePct}}%</td>
		</tr>
		{{end}}
	</table>
</body>
</html>
`
	t := template.Must(template.New("html").Parse(tpl))
	if err := t.Execute(os.Stdout, disks); err != nil {
		fmt.Println("Error generating HTML:", err)
	}
}

func normalizeDeviceID(device string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(device)
	}

	return strings.TrimRightFunc(device, func(r rune) bool {
		return r >= '0' && r <= '9'
	})
}

func getFreeColor(usage *disk.UsageStat) func(a ...interface{}) string {
	freePercent := float64(usage.Free) / float64(usage.Total) * 100

	switch {
	case freePercent < 10:
		return color.New(color.FgHiRed).SprintFunc()
	default:
		return color.New(color.FgHiGreen).SprintFunc()
	}
}
