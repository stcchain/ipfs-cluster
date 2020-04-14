package test

import (
	"encoding/hex"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	files "github.com/ipfs/go-ipfs-files"

	cid "github.com/ipfs/go-cid"
)

const shardingTestDir = "shardTesting"
const shardingTestTree = "testTree"
const shardingTestFile = "testFile"

// Variables related to adding the testing directory generated by tests
var (
	ShardingDirBalancedRootCID        = "QmdHXJgxeCFf6qDZqYYmMesV2DbZCVPEdEhj2oVTxP1y7Y"
	ShardingDirBalancedRootCIDWrapped = "QmbfGRPTUd7L1xsAZZ1A3kUFP1zkEZ9kHdb6AGaajBzGGX"
	ShardingDirTrickleRootCID         = "QmYMbx56GFNBDAaAMchtjmWjDTdqNKCSGuFxtRosiPgJL6"
	// These hashes should match all the blocks produced when adding
	// the files resulting from GetShardingDir*
	// They have been obtained by adding the "shardTesting" folder
	// to go-ipfs (with default parameters). Then doing
	// `refs -r` on the result. It contains the folder hash.
	ShardingDirCids = [28]string{
		"QmdHXJgxeCFf6qDZqYYmMesV2DbZCVPEdEhj2oVTxP1y7Y",
		"QmSpZcKTgfsxyL7nyjzTNB1gAWmGYC2t8kRPpZSG1ZbTkY",
		"QmSijPKAE61CUs57wWU2M4YxkSaRogQxYRtHoEzP2uRaQt",
		"QmYr6r514Pt8HbsFjwompLrHMyZEYg6aXfsv59Ys8uzLpr",
		"QmfEeHL3iwDE8XjeFq9HDu2B8Dfu8L94y7HUB5sh5vN9TB",
		"QmTz2gUzUNQnH3i818MAJPMLjBfRXZxoZbdNYT1K66LnZN",
		"QmPZLJ3CZYgxH4K1w5jdbAdxJynXn5TCB4kHy7u8uHC3fy",
		"QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn",
		"QmY6PArrjY66Nb4qEKWF7RUHCToRFyTsrM6cH8D6vJMSnk",
		"QmYXgh47x4gr1iL6YRqAA8RcE3XNWPfB5VJTt9dBfRnRHX",
		"QmXqkKUxgWsgXEUsxDJcs2hUrSrFnPkKyGnGdxpm1cb2me",
		"Qmbne4XHMAiZwoFYdnGrdcW3UBYA7UnFE9WoDwEjG3deZH",
		"Qmdz4kLZUjfGBSvfMxTQpcxjz2aZqupnF9KjKGpAuaZ4nT",
		"QmavW3cdGuSfYMEQiBDfobwVtPEjUnML2Ry1q8w8X3Q8Wj",
		"QmfPHRbeerRWgbu5BzxwK7UhmJGqGvZNxuFoMCUFTuhG3H",
		"QmaYNfhw7L7KWX7LYpwWt1bh6Gq2p7z1tic35PnDRnqyBf",
		"QmWWwH1GKMh6GmFQunjq7CHjr4g4z6Q4xHyDVfuZGX7MyU",
		"QmVpHQGMF5PLsvfgj8bGo9q2YyLRPMvfu1uTb3DgREFtUc",
		"QmUrdAn4Mx4kNioX9juLgwQotwFfxeo5doUNnLJrQynBEN",
		"QmdJ86B7J8mfGq6SjQy8Jz7r5x1cLcXc9M2a7T7NmSMVZx",
		"QmS77cTMdyx8P7rP2Gij6azgYPpjp2J34EVYuhB6mfjrQh",
		"QmbsBsDspFcqi7xJ4xPxcNYnduzQ5UQDw9y6trQWZGoEHq",
		"QmakAXHMeyE6fHHaeqicSKVMM2QyuGbS2g8dgUA7ns8gSY",
		"QmTC6vGbH9ABkpXfrMmYkXbxEqH12jEVGpvGzibGZEDVHK",
		"QmebQW6nfE5cPb85ZUGrSyqbFsVYwfuKsX8Ur3NWwfmnYk",
		"QmSCcsb4mNMz3CXvVjPdc7kxrx4PbitrcRN8ocmyg62oit",
		"QmZ2iUT3W7jh8QNnpWSiMZ1QYgpommCSQFZiPY5VdoCHyv",
		"QmdmUbN9JS3BK3nvcycyzFUBJqXip5zf7bdKbYM3p14e9h",
	}

	// Used for testing blockput/blockget
	ShardCid, _  = cid.Decode("zdpuAoiNm1ntWx6jpgcReTiCWFHJSTpvTw4bAAn9p6yDnznqh")
	ShardData, _ = hex.DecodeString("a16130d82a58230012209273fd63ec94bed5abb219b2d9cb010cabe4af7b0177292d4335eff50464060a")
)

// ShardingTestHelper helps generating files and folders to test adding and
// sharding in IPFS Cluster
type ShardingTestHelper struct {
	randSrc *rand.Rand
}

// NewShardingTestHelper returns a new helper.
func NewShardingTestHelper() *ShardingTestHelper {
	return &ShardingTestHelper{
		randSrc: rand.New(rand.NewSource(1)),
	}
}

// GetTreeMultiReader creates and returns a MultiFileReader for a testing
// directory tree. Files are pseudo-randomly generated and are always the same.
// Directory structure:
//   - testingTree
//     - A
//         - alpha
//             * small_file_0 (< 5 kB)
//         - beta
//             * small_file_1 (< 5 kB)
//         - delta
//             - empty
//         * small_file_2 (< 5 kB)
//         - gamma
//             * small_file_3 (< 5 kB)
//     - B
//         * medium_file (~.3 MB)
//         * big_file (3 MB)
//
// The total size in ext4 is ~3420160 Bytes = ~3340 kB = ~3.4MB
func (sth *ShardingTestHelper) GetTreeMultiReader(t *testing.T) (*files.MultiFileReader, io.Closer) {
	sf := sth.GetTreeSerialFile(t)

	mapDir := files.NewMapDirectory(map[string]files.Node{
		shardingTestTree: sf,
	})

	return files.NewMultiFileReader(mapDir, true), sf
}

// GetTreeSerialFile returns a files.Directory pointing to the testing
// directory tree (see GetTreeMultiReader).
func (sth *ShardingTestHelper) GetTreeSerialFile(t *testing.T) files.Directory {
	st := sth.makeTree(t)
	sf, err := files.NewSerialFile(sth.path(shardingTestTree), false, st)

	if err != nil {
		t.Fatal(err)
	}
	return sf.(files.Directory)
}

// GetRandFileMultiReader creates and returns a MultiFileReader for
// a testing random file of the given size (in kbs). The random
// file is different every time.
func (sth *ShardingTestHelper) GetRandFileMultiReader(t *testing.T, kbs int) (*files.MultiFileReader, io.Closer) {
	slf, sf := sth.GetRandFileReader(t, kbs)
	return files.NewMultiFileReader(slf, true), sf
}

// GetRandFileReader creates and returns a directory containing a testing
// random file of the given size (in kbs)
func (sth *ShardingTestHelper) GetRandFileReader(t *testing.T, kbs int) (files.Directory, io.Closer) {
	st := sth.makeRandFile(t, kbs)
	sf, err := files.NewSerialFile(sth.path(shardingTestFile), false, st)
	if err != nil {
		t.Fatal(err)
	}
	mapDir := files.NewMapDirectory(
		map[string]files.Node{"randomfile": sf},
	)
	return mapDir, sf
}

// Clean deletes any folder and file generated by this helper.
func (sth *ShardingTestHelper) Clean(t *testing.T) {
	err := os.RemoveAll(shardingTestDir)
	if err != nil {
		t.Fatal(err)
	}
}

func folderExists(t *testing.T, path string) bool {
	if st, err := os.Stat(path); os.IsNotExist(err) {
		return false
	} else if err != nil {
		t.Fatal(err)
	} else if !st.IsDir() {
		t.Fatalf("%s is not a directory", path)
	}
	return true
}

func makeDir(t *testing.T, path string) {
	if !folderExists(t, path) {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// see GetTreeMultiReader
func (sth *ShardingTestHelper) makeTestFolder(t *testing.T) {
	makeDir(t, shardingTestDir)
}

// This produces this:
// - shardTesting
//   - testTree
//     - A
//         - alpha
//             * small_file_0 (< 5 kB)
//         - beta
//             * small_file_1 (< 5 kB)
//         - delta
//             - empty
//         * small_file_2 (< 5 kB)
//         - gamma
//             * small_file_3 (< 5 kB)
//     - B
//         * medium_file (~.3 MB)
//         * big_file (3 MB)
//
// Take special care when modifying this function.  File data depends on order
// and each file size.  If this changes then hashes above
// recording the ipfs import hash tree must be updated manually.
func (sth *ShardingTestHelper) makeTree(t *testing.T) os.FileInfo {
	sth.makeTestFolder(t)
	basepath := sth.path(shardingTestTree)

	// do not re-create
	if folderExists(t, basepath) {
		st, _ := os.Stat(basepath)
		return st
	}

	p0 := shardingTestTree
	paths := [][]string{
		{p0, "A", "alpha"},
		{p0, "A", "beta"},
		{p0, "A", "delta", "empty"},
		{p0, "A", "gamma"},
		{p0, "B"},
	}
	for _, p := range paths {
		makeDir(t, sth.path(p...))
	}

	files := [][]string{
		{p0, "A", "alpha", "small_file_0"},
		{p0, "A", "beta", "small_file_1"},
		{p0, "A", "small_file_2"},
		{p0, "A", "gamma", "small_file_3"},
		{p0, "B", "medium_file"},
		{p0, "B", "big_file"},
	}

	fileSizes := []int{5, 5, 5, 5, 300, 3000}
	for i, fpath := range files {
		path := sth.path(fpath...)
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		sth.randFile(t, f, fileSizes[i])
		f.Sync()
		f.Close()
	}

	st, err := os.Stat(basepath)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func (sth *ShardingTestHelper) path(p ...string) string {
	paths := append([]string{shardingTestDir}, p...)
	return filepath.Join(paths...)
}

// Writes randomness to a writer up to the given size (in kBs)
func (sth *ShardingTestHelper) randFile(t *testing.T, w io.Writer, kbs int) {
	buf := make([]byte, 1024)
	for i := 0; i < kbs; i++ {
		sth.randSrc.Read(buf) // read 1 kb
		if _, err := w.Write(buf); err != nil {
			t.Fatal(err)
		}
	}
}

// this creates shardingTestFile in the testFolder. It recreates it every
// time.
func (sth *ShardingTestHelper) makeRandFile(t *testing.T, kbs int) os.FileInfo {
	sth.makeTestFolder(t)
	path := sth.path(shardingTestFile)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	defer f.Sync()
	sth.randFile(t, f, kbs)
	st, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	return st

}
