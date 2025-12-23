package main

import (
	"log"
	"os"

	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/schema"
)

type TestData struct {
	Id    int64   `parquet:"name=id, type=INT64"`
	Name  string  `parquet:"name=name, type=BYTE_ARRAY, converted=UTF8"`
	Value float64 `parquet:"name=value, type=DOUBLE"`
}

func main() {
	sc, err := schema.NewSchemaFromStruct(TestData{})
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create("test.parquet")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	writer := file.NewParquetWriter(f, sc.Root())
	defer writer.Close()

	rgw := writer.AppendRowGroup()

	idWriter, _ := rgw.NextColumn()
	idWriter.(*file.Int64ColumnChunkWriter).WriteBatch([]int64{1, 2, 3}, nil, nil)

	nameWriter, _ := rgw.NextColumn()
	nameWriter.(*file.ByteArrayColumnChunkWriter).WriteBatch([]parquet.ByteArray{
		parquet.ByteArray("Alice"),
		parquet.ByteArray("Bob"),
		parquet.ByteArray("Charlie"),
	}, nil, nil)

	valueWriter, _ := rgw.NextColumn()
	valueWriter.(*file.Float64ColumnChunkWriter).WriteBatch([]float64{10.5, 20.7, 30.1}, nil, nil)
}
