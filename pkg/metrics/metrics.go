package metrics

import (
    "bufio"
    "fmt"
    "strconv"
    "strings"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "go.bug.st/serial"
    "go.uber.org/zap"
)

// Prefix för alla metriker
const metricPrefix = "elcentral_"

// Metrics definitioner
var (
    // Energimätvärden
    activeEnergy = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: metricPrefix + "active_energy_kwh",
            Help: "Cumulative active energy in kWh",
        },
        []string{"direction", "tariff"},
    )
    reactiveEnergy = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: metricPrefix + "reactive_energy_kvarh",
            Help: "Cumulative reactive energy in kvarh",
        },
        []string{"direction", "tariff"},
    )

    // Effektmätvärden
    activePower = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: metricPrefix + "active_power_kw",
            Help: "Current active power in kW",
        },
        []string{"direction"},
    )
    reactivePower = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: metricPrefix + "reactive_power_kvar",
            Help: "Current reactive power in kvar",
        },
        []string{"direction"},
    )

    // Omedelbar effekt per fas
    instantaneousActivePower = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: metricPrefix + "instantaneous_active_power_kw",
            Help: "Instantaneous active power in kW",
        },
        []string{"phase"},
    )
    instantaneousReactivePower = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: metricPrefix + "instantaneous_reactive_power_kvar",
            Help: "Instantaneous reactive power in kvar",
        },
        []string{"phase"},
    )

    // Spänning och ström per fas
    voltage = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: metricPrefix + "voltage_v",
            Help: "Phase voltage in volts",
        },
        []string{"phase"},
    )
    current = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: metricPrefix + "current_a",
            Help: "Phase current in amperes",
        },
        []string{"phase"},
    )

    // Metadata
    meterID = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: metricPrefix + "meter_id",
            Help: "Meter identification string (Numerisk representation)",
        },
        []string{"device"},
    )
    timestamp = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: metricPrefix + "timestamp",
            Help: "Meter timestamp as Unix epoch",
        },
        []string{"device"},
    )
)

// init registrerar alla metriker med Prometheus
func init() {
    // Energimätvärden
    prometheus.MustRegister(activeEnergy)
    prometheus.MustRegister(reactiveEnergy)

    // Effektmätvärden
    prometheus.MustRegister(activePower)
    prometheus.MustRegister(reactivePower)

    // Omedelbar effekt per fas
    prometheus.MustRegister(instantaneousActivePower)
    prometheus.MustRegister(instantaneousReactivePower)

    // Spänning och ström per fas
    prometheus.MustRegister(voltage)
    prometheus.MustRegister(current)

    // Metadata
    prometheus.MustRegister(meterID)
    prometheus.MustRegister(timestamp)
}

// ReadSerialData läser data från den seriella porten och vidarebefordrar den till parseAndSetMetrics
func ReadSerialData(serialPort serial.Port, sugar *zap.SugaredLogger) {
    reader := bufio.NewScanner(serialPort)

    for reader.Scan() {
        line := reader.Text()
        sugar.Debugf("Received data: %s", line)
        parseAndSetMetrics(line, sugar)
    }

    if err := reader.Err(); err != nil {
        sugar.Errorf("Error reading from serial port: %v", err)
    }
}

// parseAndSetMetrics tolkar en rad data och sätter motsvarande Prometheus-metrik
func parseAndSetMetrics(data string, sugar *zap.SugaredLogger) {
    lines := strings.Split(data, "\n")

    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }

        sugar.Debugf("Processing line: %s", line)

        switch {
        // Metadata
        case strings.HasPrefix(line, "0-0:96.1.0("):
            value := extractValueText(line)
            sugar.Debugf("Setting meter_id: %s", value)
            hashedID := hashString(value)
            meterID.WithLabelValues("meter").Set(float64(hashedID))

        case strings.HasPrefix(line, "0-0:1.0.0("):
            value := extractValueText(line)
            sugar.Debugf("Setting timestamp: %s", value)
            cleanedValue := strings.TrimSuffix(value, "W")
            unixTs := parseTimestamp(cleanedValue)
            timestamp.WithLabelValues("meter").Set(float64(unixTs))

        // Energimätvärden
        case strings.HasPrefix(line, "1-0:1.8.0("):
            value := extractValue(line)
            sugar.Debugf("Setting active_energy_kwh(import, T1): %f", value)
            activeEnergy.WithLabelValues("import", "T1").Set(value)

        case strings.HasPrefix(line, "1-0:2.8.0("):
            value := extractValue(line)
            sugar.Debugf("Setting active_energy_kwh(export, T1): %f", value)
            activeEnergy.WithLabelValues("export", "T1").Set(value)

        case strings.HasPrefix(line, "1-0:3.8.0("):
            value := extractValue(line)
            sugar.Debugf("Setting reactive_energy_kvarh(import, T1): %f", value)
            reactiveEnergy.WithLabelValues("import", "T1").Set(value)

        case strings.HasPrefix(line, "1-0:4.8.0("):
            value := extractValue(line)
            sugar.Debugf("Setting reactive_energy_kvarh(export, T1): %f", value)
            reactiveEnergy.WithLabelValues("export", "T1").Set(value)

        // Effektmätvärden
        case strings.HasPrefix(line, "1-0:1.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting active_power_kw(import): %f", value)
            activePower.WithLabelValues("import").Set(value)

        case strings.HasPrefix(line, "1-0:2.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting active_power_kw(export): %f", value)
            activePower.WithLabelValues("export").Set(value)

        case strings.HasPrefix(line, "1-0:3.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting reactive_power_kvar(import): %f", value)
            reactivePower.WithLabelValues("import").Set(value)

        case strings.HasPrefix(line, "1-0:4.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting reactive_power_kvar(export): %f", value)
            reactivePower.WithLabelValues("export").Set(value)

        // Instantaneous Active Power per phase
        case strings.HasPrefix(line, "1-0:21.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting instantaneous_active_power_kw(L1): %f", value)
            instantaneousActivePower.WithLabelValues("L1").Set(value)

        case strings.HasPrefix(line, "1-0:22.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting instantaneous_active_power_kw(L1_export): %f", value)
            instantaneousActivePower.WithLabelValues("L1_export").Set(value)

        case strings.HasPrefix(line, "1-0:41.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting instantaneous_active_power_kw(L2): %f", value)
            instantaneousActivePower.WithLabelValues("L2").Set(value)

        case strings.HasPrefix(line, "1-0:42.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting instantaneous_active_power_kw(L2_export): %f", value)
            instantaneousActivePower.WithLabelValues("L2_export").Set(value)

        case strings.HasPrefix(line, "1-0:61.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting instantaneous_active_power_kw(L3): %f", value)
            instantaneousActivePower.WithLabelValues("L3").Set(value)

        case strings.HasPrefix(line, "1-0:62.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting instantaneous_active_power_kw(L3_export): %f", value)
            instantaneousActivePower.WithLabelValues("L3_export").Set(value)

        // Instantaneous Reactive Power per phase
        case strings.HasPrefix(line, "1-0:23.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting instantaneous_reactive_power_kvar(L1): %f", value)
            instantaneousReactivePower.WithLabelValues("L1").Set(value)

        case strings.HasPrefix(line, "1-0:24.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting instantaneous_reactive_power_kvar(L1_export): %f", value)
            instantaneousReactivePower.WithLabelValues("L1_export").Set(value)

        case strings.HasPrefix(line, "1-0:43.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting instantaneous_reactive_power_kvar(L2): %f", value)
            instantaneousReactivePower.WithLabelValues("L2").Set(value)

        case strings.HasPrefix(line, "1-0:44.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting instantaneous_reactive_power_kvar(L2_export): %f", value)
            instantaneousReactivePower.WithLabelValues("L2_export").Set(value)

        case strings.HasPrefix(line, "1-0:63.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting instantaneous_reactive_power_kvar(L3): %f", value)
            instantaneousReactivePower.WithLabelValues("L3").Set(value)

        case strings.HasPrefix(line, "1-0:64.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting instantaneous_reactive_power_kvar(L3_export): %f", value)
            instantaneousReactivePower.WithLabelValues("L3_export").Set(value)

        // Spänning per fas
        case strings.HasPrefix(line, "1-0:32.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting voltage_v(L1): %f", value)
            voltage.WithLabelValues("L1").Set(value)

        case strings.HasPrefix(line, "1-0:52.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting voltage_v(L2): %f", value)
            voltage.WithLabelValues("L2").Set(value)

        case strings.HasPrefix(line, "1-0:72.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting voltage_v(L3): %f", value)
            voltage.WithLabelValues("L3").Set(value)

        // Ström per fas
        case strings.HasPrefix(line, "1-0:31.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting current_a(L1): %f", value)
            current.WithLabelValues("L1").Set(value)

        case strings.HasPrefix(line, "1-0:51.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting current_a(L2): %f", value)
            current.WithLabelValues("L2").Set(value)

        case strings.HasPrefix(line, "1-0:71.7.0("):
            value := extractValue(line)
            sugar.Debugf("Setting current_a(neutral): %f", value)
            current.WithLabelValues("L3").Set(value)

        // Hantera ogiltiga eller specifika linjer
        case strings.HasPrefix(line, "/"):
            sugar.Debugf("Ignoring non-OBIS line: %s", line)

        case strings.HasPrefix(line, "!"):
            sugar.Debugf("Ignoring termination line: %s", line)

        default:
            sugar.Warnf("Unhandled OBIS code in line: %s", line)
        }
    }
}

// extractValue extraherar float64-värdet från en linje med OBIS-kod
func extractValue(line string) float64 {
    start := strings.Index(line, "(") + 1
    end := strings.Index(line, "*")
    if start < 0 || end < 0 || end <= start {
        // För vissa OBIS-koder saknas enhet
        end = strings.Index(line, ")")
        if end <= start {
            return 0
        }
    }
    var value float64
    fmt.Sscanf(line[start:end], "%f", &value)
    return value
}

// extractValueText extraherar textvärdet från en linje med OBIS-kod
func extractValueText(line string) string {
    start := strings.Index(line, "(") + 1
    end := strings.Index(line, ")")
    if start < 0 || end < 0 || end <= start {
        return ""
    }
    return line[start:end]
}

// hashString genererar en enkel hash från en sträng (för meter ID)
func hashString(s string) int {
    hash := 0
    for _, c := range s {
        hash = hash*31 + int(c)
    }
    return hash
}

// parseTimestamp omvandlar en tidssträng till Unix epoch
func parseTimestamp(ts string) int {
    if len(ts) < 12 {
        return 0
    }
    year, _ := strconv.Atoi("20" + ts[0:2])
    month, _ := strconv.Atoi(ts[2:4])
    day, _ := strconv.Atoi(ts[4:6])
    hour, _ := strconv.Atoi(ts[6:8])
    minute, _ := strconv.Atoi(ts[8:10])
    second, _ := strconv.Atoi(ts[10:12])

    t := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
    return int(t.Unix())
}