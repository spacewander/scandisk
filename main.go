package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

const (
	PathSeparator = string(filepath.Separator)
)

var (
	OS string = runtime.GOOS

	// arguments
	rootDir        string
	output         string
	outputFilename string

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

func printErrorThenExit(fmtStr string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, fmtStr+"\n", err)
	} else {
		fmt.Fprintf(os.Stderr, fmtStr+"\n")
	}
	os.Exit(1)
}

func travelDir(root *Node, dir string, depth int) {
	if OS == "linux" && depth < 2 {
		if strings.HasPrefix(dir, "/proc") {
			return
		}
		depth += 1
	}
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
				travelDir(&fileNode, file, depth)
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

func printNode(node *Node, depth int) {
	fmt.Println(strings.Repeat("  ", depth) + node.String())
	for _, node := range node.children {
		printNode(&node, depth+1)
	}
}

func displayAsText(root *Node) {
	printNode(root, 0)
}

type NodeJsonDump struct {
	Children []NodeJsonDump `json:"children"`
	Text     string         `json:"text"`
}

func dumpNodeToJson(node *Node) NodeJsonDump {
	dump := NodeJsonDump{}
	dump.Text = node.String()
	for _, child := range node.children {
		dump.Children = append(dump.Children, dumpNodeToJson(&child))
	}
	return dump
}

func releaseStaticAssets(assets []string) error {
	for _, fn := range assets {
		data, err := Asset("assets/" + fn)
		if err != nil {
			return err
		}
		if err = ioutil.WriteFile(fn, data, 0644); err != nil {
			return err
		}
	}
	return nil
}

func displayAsHtml(root *Node) {
	jsonData := dumpNodeToJson(root)
	result, err := json.Marshal(jsonData)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dump json data failed: %v", err)
	} else {
		tmplData, err := Asset("assets/scandisk.tmpl")
		if err != nil {
			printErrorThenExit("template file not found: %v", err)
		}
		tmpl, err := template.New("scandisk").Parse(string(tmplData))
		if err != nil {
			printErrorThenExit("parse template failed: %v", err)
		}
		if !strings.HasSuffix(outputFilename, ".html") {
			outputFilename += ".html"
		}
		file, err := os.OpenFile(outputFilename,
			os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			printErrorThenExit("open "+outputFilename+" failed: %v", err)
		}
		defer file.Close()
		err = tmpl.Execute(file, struct {
			Data string
		}{
			string(result),
		})
		if err != nil {
			printErrorThenExit("render template failed: %v", err)
		}

		err = releaseStaticAssets([]string{
			"jquery.1.12.1.min.js", "jstree.min.js", "style.min.css",
			"throbber.gif", "32px.png", "40px.png",
		})
		if err != nil {
			printErrorThenExit("release static assets failed: %v", err)
		}
	}
}

func main() {
	flag.StringVar(&rootDir, "root", "/", "Set the directory to scan from")
	flag.StringVar(&output, "output", "html",
		"Set the output destination(html or text). It will output to '$filename.html' by default.")
	flag.StringVar(&outputFilename, "filename", "output",
		"If the output is set to html, this argument decides the output file name.")
	flag.Parse()
	if info, err := os.Stat(rootDir); err != nil || !info.IsDir() {
		printErrorThenExit("The root argument should be a directory", nil)
	}
	BlockSize = statfs(rootDir)
	rootNode := Node{
		name: rootDir,
	}
	travelDir(&rootNode, rootDir, 0)
	switch output {
	case "text":
		displayAsText(&rootNode)
	default:
		displayAsHtml(&rootNode)
	}
}
