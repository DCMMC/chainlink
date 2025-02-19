package main

import (
	"fmt"
	"os"
	"path/filepath"

	gethParams "github.com/ethereum/go-ethereum/params"
	"github.com/DCMMC/chainlink/core/internal/gethwrappers"
)

func main() {
	abiPath := os.Args[1]
	binPath := os.Args[2]
	className := os.Args[3]
	pkgName := os.Args[4]
	fmt.Println("Generating", pkgName, "contract wrapper")

	cwd, err := os.Getwd() // gethwrappers directory
	if err != nil {
		gethwrappers.Exit("could not get working directory", err)
	}
	outDir := filepath.Join(cwd, "generated", pkgName)
	if mkdErr := os.MkdirAll(outDir, 0700); err != nil {
		gethwrappers.Exit("failed to create wrapper dir", mkdErr)
	}
	outPath := filepath.Join(outDir, pkgName+".go")

	gethwrappers.Abigen(gethwrappers.AbigenArgs{
		Bin: binPath, ABI: abiPath, Out: outPath, Type: className, Pkg: pkgName,
	})

	// Build succeeded, so update the versions db with the new contract data
	versions, err := gethwrappers.ReadVersionsDB()
	if err != nil {
		gethwrappers.Exit("could not read current versions database", err)
	}
	versions.GethVersion = gethParams.Version
	versions.ContractVersions[pkgName] = gethwrappers.ContractVersion{
		Hash:       gethwrappers.VersionHash(abiPath, binPath),
		AbiPath:    abiPath,
		BinaryPath: binPath,
	}
	if err := gethwrappers.WriteVersionsDB(versions); err != nil {
		gethwrappers.Exit("could not save versions db", err)
	}
}
