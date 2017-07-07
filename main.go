package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	PathSeparator = string(filepath.Separator)
)

var (
	BlockSize int64
)

type Node struct {
	size     int64
	name     string
	children []Node
}

var validUnits = []struct {
	symbol     string
	multiplier int64
}{
	// use 1024 because the block size is the multiplier of 1024
	{"GB", 1024 * 1024 * 1024},
	{"MB", 1024 * 1024},
	{"KB", 1024},
}

// Thanks for https://gist.github.com/DavidVaini/10308388
func Round(val float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= 0.5 {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func int64ToSizeStr(size int64) string {
	for _, unit := range validUnits {
		if unit.multiplier <= size {
			roundedSize := Round(float64(size)/float64(unit.multiplier), 2)
			return strconv.FormatFloat(roundedSize, 'f', -1, 64) + " " + unit.symbol
		}
	}
	return strconv.FormatInt(size, 10) + "  B"
}

func (n *Node) String() string {
	return fmt.Sprintf("%-10s %s", int64ToSizeStr(n.size), n.name)
}

func travelDir(root *Node, dir string) {
	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return
	}
	for _, file := range files {
		_, name := filepath.Split(file)
		fileNode := Node{
			name: name,
		}
		if info, err := os.Lstat(file); err == nil {
			if info.IsDir() {
				fileNode.name += PathSeparator
				travelDir(&fileNode, file)
			} else {
				// file system stores data in blocks, so the disk occupation
				// should be the multiplier of block size.
				blocks := math.Ceil(float64(info.Size()) / float64(BlockSize))
				fileNode.size = int64(blocks) * BlockSize
			}
			root.size += fileNode.size
		}
		root.children = append(root.children, fileNode)
	}
	// To use this API, go1.8+ is in need
	sort.Slice(root.children, func(i, j int) bool {
		return root.children[i].size > root.children[j].size
	})
}

func printNode(node *Node, level int) {
	fmt.Println(strings.Repeat("  ", level) + node.String())
	for _, node := range node.children {
		printNode(&node, level+1)
	}
}

func displayAsText(root *Node) {
	printNode(root, 0)
}

func main() {
	var root string
	flag.StringVar(&root, "root", "/", "Set the directory to scan from")
	flag.Parse()
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "The root %s should be a directory\n", root)
		os.Exit(1)
	}
	BlockSize = statfs(root)
	rootNode := Node{
		name: root,
	}
	travelDir(&rootNode, root)
	displayAsText(&rootNode)
}
