// +build ignore

package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
)

func main() {
	// Generate 2 days of data (today and yesterday) at 5-minute intervals
	// End time is now, start time is 2 days ago
	endTime := time.Now().UTC()
	startTime := endTime.Add(-2 * 24 * time.Hour)
	interval := 5 * time.Minute
	numPoints := int((2 * 24 * time.Hour) / interval)

	pool := memory.NewGoAllocator()

	// Build timestamp column
	tsBuilder := array.NewTimestampBuilder(pool, &arrow.TimestampType{Unit: arrow.Millisecond, TimeZone: "UTC"})
	defer tsBuilder.Release()

	// Build metric columns
	cpuBuilder := array.NewFloat64Builder(pool)
	defer cpuBuilder.Release()

	memoryBuilder := array.NewFloat64Builder(pool)
	defer memoryBuilder.Release()

	diskBuilder := array.NewFloat64Builder(pool)
	defer diskBuilder.Release()

	networkBuilder := array.NewFloat64Builder(pool)
	defer networkBuilder.Release()

	requestsBuilder := array.NewInt64Builder(pool)
	defer requestsBuilder.Release()

	errorsBuilder := array.NewInt64Builder(pool)
	defer errorsBuilder.Release()

	hostBuilder := array.NewStringBuilder(pool)
	defer hostBuilder.Release()

	hosts := []string{"server-1", "server-2", "server-3"}
	rand.Seed(time.Now().UnixNano())

	// Generate data for each host
	for _, host := range hosts {
		currentTime := startTime
		baseCPU := 20.0 + rand.Float64()*30
		baseMemory := 40.0 + rand.Float64()*20
		baseDisk := 50.0 + rand.Float64()*20

		for i := 0; i < numPoints; i++ {
			tsBuilder.Append(arrow.Timestamp(currentTime.UnixMilli()))

			// Simulate realistic patterns with daily cycles
			hour := currentTime.Hour()
			dayFactor := 1.0
			if hour >= 9 && hour <= 17 {
				dayFactor = 1.5 // Higher load during business hours
			}

			// Add some noise and trends
			cpu := baseCPU*dayFactor + math.Sin(float64(i)/100)*10 + rand.Float64()*15
			cpu = math.Max(0, math.Min(100, cpu))
			cpuBuilder.Append(cpu)

			memory := baseMemory + math.Sin(float64(i)/200)*5 + rand.Float64()*10
			memory = math.Max(0, math.Min(100, memory))
			memoryBuilder.Append(memory)

			disk := baseDisk + float64(i)/float64(numPoints)*10 + rand.Float64()*2
			disk = math.Max(0, math.Min(100, disk))
			diskBuilder.Append(disk)

			network := 100*dayFactor + rand.Float64()*50
			networkBuilder.Append(network)

			requests := int64(1000*dayFactor + rand.Float64()*500)
			requestsBuilder.Append(requests)

			errors := int64(rand.Float64() * 10)
			errorsBuilder.Append(errors)

			hostBuilder.Append(host)

			currentTime = currentTime.Add(interval)
		}
	}

	// Create Arrow arrays
	tsArray := tsBuilder.NewArray()
	defer tsArray.Release()
	cpuArray := cpuBuilder.NewArray()
	defer cpuArray.Release()
	memoryArray := memoryBuilder.NewArray()
	defer memoryArray.Release()
	diskArray := diskBuilder.NewArray()
	defer diskArray.Release()
	networkArray := networkBuilder.NewArray()
	defer networkArray.Release()
	requestsArray := requestsBuilder.NewArray()
	defer requestsArray.Release()
	errorsArray := errorsBuilder.NewArray()
	defer errorsArray.Release()
	hostArray := hostBuilder.NewArray()
	defer hostArray.Release()

	// Create schema
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "timestamp", Type: &arrow.TimestampType{Unit: arrow.Millisecond, TimeZone: "UTC"}},
		{Name: "cpu_percent", Type: arrow.PrimitiveTypes.Float64},
		{Name: "memory_percent", Type: arrow.PrimitiveTypes.Float64},
		{Name: "disk_percent", Type: arrow.PrimitiveTypes.Float64},
		{Name: "network_mbps", Type: arrow.PrimitiveTypes.Float64},
		{Name: "requests", Type: arrow.PrimitiveTypes.Int64},
		{Name: "errors", Type: arrow.PrimitiveTypes.Int64},
		{Name: "host", Type: arrow.BinaryTypes.String},
	}, nil)

	// Create record batch
	record := array.NewRecord(schema, []arrow.Array{
		tsArray, cpuArray, memoryArray, diskArray, networkArray, requestsArray, errorsArray, hostArray,
	}, int64(tsArray.Len()))
	defer record.Release()

	// Write to parquet file
	file, err := os.Create("metrics_timeseries.parquet")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	writer, err := pqarrow.NewFileWriter(schema, file, parquet.NewWriterProperties(), pqarrow.DefaultWriterProps())
	if err != nil {
		fmt.Printf("Error creating writer: %v\n", err)
		os.Exit(1)
	}

	if err := writer.Write(record); err != nil {
		fmt.Printf("Error writing record: %v\n", err)
		os.Exit(1)
	}

	if err := writer.Close(); err != nil {
		fmt.Printf("Error closing writer: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created metrics_timeseries.parquet with %d rows (3 hosts x %d points)\n",
		tsArray.Len(), numPoints)
}
