package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type TelephonyInfo struct {
	Operator   string `json:"operator"`
	NetType    string `json:"net_type"`
	SignalBars int    `json:"signal_bars"`
	Dbm        int    `json:"dbm"`
	SignalText string `json:"signal_text"`
}

func getTelephonyInfo() TelephonyInfo {
	info := TelephonyInfo{
		Operator:   "Tidak Ada SIM",
		NetType:    "Tidak Ada Jaringan",
		SignalBars: 0,
		Dbm:        0,
		SignalText: "Tidak Ada Sinyal",
	}

	// 1. Get Operator
	outSimOp, _ := exec.Command("getprop", "gsm.sim.operator.alpha").Output()
	outOpAlpha, _ := exec.Command("getprop", "gsm.operator.alpha").Output()
	outOpNum, _ := exec.Command("getprop", "gsm.operator.numeric").Output()

	rawSim := strings.TrimSpace(string(outSimOp))
	rawAlpha := strings.TrimSpace(string(outOpAlpha))
	rawNum := strings.TrimSpace(string(outOpNum))

	// Normalisasi Operator
	candidates := []string{}
	for _, s := range strings.Split(rawSim, ",") {
		if t := strings.TrimSpace(s); t != "" && !strings.EqualFold(t, "unknown") {
			candidates = append(candidates, t)
		}
	}
	for _, s := range strings.Split(rawAlpha, ",") {
		if t := strings.TrimSpace(s); t != "" && !strings.EqualFold(t, "unknown") {
			candidates = append(candidates, t)
		}
	}

	opName := ""
	if len(candidates) > 0 {
		opName = candidates[0]
	}

	// Rapikan nama operator
	lowerOp := strings.ToLower(opName)
	if strings.Contains(lowerOp, "smartfren") {
		info.Operator = "Smartfren"
	} else if strings.Contains(lowerOp, "telkomsel") || strings.Contains(lowerOp, "tsel") || strings.Contains(lowerOp, "by.u") {
		info.Operator = "Telkomsel"
	} else if strings.Contains(lowerOp, "indosat") || strings.Contains(lowerOp, "im3") || strings.Contains(lowerOp, "ooredoo") {
		info.Operator = "Indosat Ooredoo"
	} else if strings.Contains(lowerOp, "xl") || strings.Contains(lowerOp, "axiata") {
		info.Operator = "XL Axiata"
	} else if strings.Contains(lowerOp, "tri") || strings.Contains(lowerOp, "three") || strings.Contains(lowerOp, "3") {
		info.Operator = "Tri (3)"
	} else if opName != "" {
		info.Operator = strings.Title(strings.ToLower(opName))
	} else if strings.HasPrefix(rawNum, "51009") || strings.HasPrefix(rawNum, "51028") {
		info.Operator = "Smartfren"
	} else if strings.HasPrefix(rawNum, "51010") {
		info.Operator = "Telkomsel"
	} else if strings.HasPrefix(rawNum, "51001") || strings.HasPrefix(rawNum, "51021") {
		info.Operator = "Indosat Ooredoo"
	} else if strings.HasPrefix(rawNum, "51011") {
		info.Operator = "XL Axiata"
	} else if strings.HasPrefix(rawNum, "51089") {
		info.Operator = "Tri (3)"
	}

	// 2. Get Network Type (RAT)
	outNetType, _ := exec.Command("getprop", "gsm.network.type").Output()
	rawNet := strings.TrimSpace(string(outNetType))
	for _, n := range strings.Split(rawNet, ",") {
		t := strings.TrimSpace(n)
		if t != "" && !strings.EqualFold(t, "unknown") {
			upper := strings.ToUpper(t)
			if strings.Contains(upper, "LTE") {
				info.NetType = "4G LTE"
			} else if strings.Contains(upper, "NR") {
				info.NetType = "5G NR"
			} else if strings.Contains(upper, "HSPA") || strings.Contains(upper, "UMTS") || strings.Contains(upper, "HSDPA") {
				info.NetType = "3G HSPA"
			} else if strings.Contains(upper, "GSM") || strings.Contains(upper, "EDGE") || strings.Contains(upper, "GPRS") {
				info.NetType = "2G EDGE"
			} else {
				info.NetType = upper
			}
			break
		}
	}

	// 3. Get Signal Strength via dumpsys telephony.registry
	outDump, err := exec.Command("dumpsys", "telephony.registry").Output()
	if err == nil {
		dumpStr := string(outDump)
		lines := strings.Split(dumpStr, "\n")
		reLte := regexp.MustCompile(`mLte=CellSignalStrengthLte:\s*rssi=(-?\d+)\s+rsrp=(-?\d+)[^}]*level=(\d+)`)
		reNr := regexp.MustCompile(`mNr=CellSignalStrengthNr:\s*\{[^}]*ssRsrp\s*=\s*(-?\d+)[^}]*level\s*=\s*(\d+)`)

		for _, l := range lines {
			if !strings.Contains(l, "mSignalStrength=SignalStrength:{") {
				continue
			}

			// Prioritaskan NR jika ada
			if matchNr := reNr.FindStringSubmatch(l); len(matchNr) > 2 {
				rsrp, _ := strconv.Atoi(matchNr[1])
				lvl, _ := strconv.Atoi(matchNr[2])
				if rsrp < 0 && rsrp != 2147483647 {
					info.Dbm = rsrp
					info.SignalBars = lvl
					if info.NetType == "Tidak Ada Jaringan" {
						info.NetType = "5G NR"
					}
					break
				}
			}

			// Cek LTE
			if matchLte := reLte.FindStringSubmatch(l); len(matchLte) > 3 {
				rssi, _ := strconv.Atoi(matchLte[1])
				rsrp, _ := strconv.Atoi(matchLte[2])
				lvl, _ := strconv.Atoi(matchLte[3])

				if (rsrp < 0 && rsrp != 2147483647) || (rssi < 0 && rssi != 2147483647) {
					if rsrp < 0 && rsrp != 2147483647 {
						info.Dbm = rsrp
					} else {
						info.Dbm = rssi
					}
					info.SignalBars = lvl
					if info.NetType == "Tidak Ada Jaringan" {
						info.NetType = "4G LTE"
					}
					break
				}
			}
		}
	}

	// 4. Format SignalText
	if info.NetType != "Tidak Ada Jaringan" {
		if info.Dbm != 0 && info.SignalBars > 0 {
			info.SignalText = fmt.Sprintf("%s · %d/4 Bar (%d dBm)", info.NetType, info.SignalBars, info.Dbm)
		} else if info.SignalBars > 0 {
			info.SignalText = fmt.Sprintf("%s · %d/4 Bar", info.NetType, info.SignalBars)
		} else if info.Dbm != 0 {
			info.SignalText = fmt.Sprintf("%s (%d dBm)", info.NetType, info.Dbm)
		} else {
			info.SignalText = info.NetType
		}
	}

	return info
}

func main() {
	info := getTelephonyInfo()
	fmt.Printf("Operator:    %s\n", info.Operator)
	fmt.Printf("Network:     %s\n", info.NetType)
	fmt.Printf("Signal Bars: %d\n", info.SignalBars)
	fmt.Printf("dBm:         %d\n", info.Dbm)
	fmt.Printf("Signal Text: %s\n", info.SignalText)
}
