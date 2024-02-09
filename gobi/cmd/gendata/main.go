// Command gendata writes the sample .dbf tables under demos/.
//
//	go run ./gobi/cmd/gendata
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func main() {
	out, err := demosDir()
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fatal(err)
	}

	y, m, d := byte(time.Now().Year()-1900), byte(time.Now().Month()), byte(time.Now().Day())

	mustWrite(filepath.Join(out, "customers.dbf"), []dbf.Field{
		{Name: "CODE", Type: 'C', Length: 4},
		{Name: "NAME", Type: 'C', Length: 20},
		{Name: "CITY", Type: 'C', Length: 15},
		{Name: "BALANCE", Type: 'N', Length: 10, Decimals: 2},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "C001"}, {Str: "Ada Lovelace"}, {Str: "London"}, {Num: 120.50}}},
		{Values: []dbf.Value{{Str: "C002"}, {Str: "Alan Turing"}, {Str: "Manchester"}, {Num: 45.00}}},
		{Values: []dbf.Value{{Str: "C003"}, {Str: "Grace Hopper"}, {Str: "New York"}, {Num: 310.25}}},
		{Values: []dbf.Value{{Str: "C004"}, {Str: "Katherine Johnson"}, {Str: "Hampton"}, {Num: 0}}},
		{Values: []dbf.Value{{Str: "C005"}, {Str: "Donald Knuth"}, {Str: "Stanford"}, {Num: 88.10}}},
	}, y, m, d)

	mustWrite(filepath.Join(out, "orders.dbf"), []dbf.Field{
		{Name: "ORD_ID", Type: 'C', Length: 5},
		{Name: "CODE", Type: 'C', Length: 4},
		{Name: "AMOUNT", Type: 'N', Length: 10, Decimals: 2},
		{Name: "ODATE", Type: 'D', Length: 8},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "O1001"}, {Str: "C001"}, {Num: 50.00}, {Str: "19860704"}}},
		{Values: []dbf.Value{{Str: "O1002"}, {Str: "C001"}, {Num: 70.50}, {Str: "19860815"}}},
		{Values: []dbf.Value{{Str: "O1003"}, {Str: "C002"}, {Num: 45.00}, {Str: "19860901"}}},
		{Values: []dbf.Value{{Str: "O1004"}, {Str: "C003"}, {Num: 200.00}, {Str: "19861012"}}},
		{Values: []dbf.Value{{Str: "O1005"}, {Str: "C003"}, {Num: 110.25}, {Str: "19861120"}}},
		{Values: []dbf.Value{{Str: "O1006"}, {Str: "C005"}, {Num: 88.10}, {Str: "19861205"}}},
	}, y, m, d)

	mustWrite(filepath.Join(out, "products.dbf"), []dbf.Field{
		{Name: "SKU", Type: 'C', Length: 6},
		{Name: "DESCR", Type: 'C', Length: 24},
		{Name: "PRICE", Type: 'N', Length: 8, Decimals: 2},
		{Name: "QTY", Type: 'N', Length: 5, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "BK0001"}, {Str: "Analytical Engine notes"}, {Num: 29.90}, {Num: 12}}},
		{Values: []dbf.Value{{Str: "BK0002"}, {Str: "On Computable Numbers"}, {Num: 34.50}, {Num: 8}}},
		{Values: []dbf.Value{{Str: "BK0003"}, {Str: "COBOL primer"}, {Num: 19.95}, {Num: 25}}},
		{Values: []dbf.Value{{Str: "BK0004"}, {Str: "Orbital mechanics"}, {Num: 42.00}, {Num: 5}}},
		{Values: []dbf.Value{{Str: "BK0005"}, {Str: "The Art of Computer Prog"}, {Num: 99.00}, {Num: 3}}},
	}, y, m, d)

	fmt.Println("wrote customers.dbf, orders.dbf, products.dbf in", out)
}

// demosDir is demos/ next to go.mod, so the command works from any working directory.
func demosDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return filepath.Join(dir, "demos"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}

func mustWrite(path string, fields []dbf.Field, recs []dbf.Record, y, m, d byte) {
	_ = os.Remove(path)
	f, err := os.Create(path)
	if err != nil {
		fatal(err)
	}
	tab, err := dbf.Create(f, fields, y, m, d)
	if err != nil {
		_ = f.Close()
		fatal(err)
	}
	for _, r := range recs {
		if err := tab.Append(r); err != nil {
			_ = f.Close()
			fatal(err)
		}
	}
	if err := tab.Close(); err != nil {
		_ = f.Close()
		fatal(err)
	}
	if err := f.Close(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
