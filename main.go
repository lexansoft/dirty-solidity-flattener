package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var processed map[string]bool = make(map[string]bool)
var spdx_inserted bool = false

func processFile(base_dir string, file_name string, out *bufio.Writer) {

	once := false
	file_name = filepath.Clean(file_name)

	if processed[file_name] {
		return
	}
	processed[file_name] = true

	fmt.Printf("Processing file: %s\n", file_name)

	file, err := os.Open(file_name)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	importPattern := regexp.MustCompile(`^\s*import\s+"([^"]+)";$`)
	spdxPattern := regexp.MustCompile(`^\s*\/\/\s*SPDX-License-Identifier: ([^$]+)$`)

	var all_lines bytes.Buffer
	var al_writer = &all_lines

	for scanner.Scan() {
		line := scanner.Text()

		if importPattern.MatchString(line) {
			// Extract the imported file name
			match := importPattern.FindStringSubmatch(line)
			if len(match) > 1 {
				importedFileName := match[1]

				dir := filepath.Dir(file_name)
				name_to_use := importedFileName
				_, err := os.Stat(name_to_use)

				if err != nil {
					name_to_use = dir + "/" + importedFileName
					_, err := os.Stat(name_to_use)
					if err != nil {
						name_to_use = dir + "/node_modules/" + importedFileName
						_, err := os.Stat(name_to_use)
						if err != nil {
							name_to_use = base_dir + "/node_modules/" + importedFileName
							_, err := os.Stat(name_to_use)
							if err != nil {
								panic("Could not find file: " + importedFileName)
							}
						}
					}

				}

				if base_dir == "" {
					base_dir = dir
				}

				processFile(base_dir, name_to_use, out)
			} else {
				panic("Invalid import statement: " + line)
			}
		} else if spdxPattern.MatchString(line) {
			if !spdx_inserted {
				if !once {
					al_writer.WriteString("// File: " + path.Base(file_name) + "\n")
					once = true
				}
				spdx_inserted = true
				al_writer.WriteString(line + "\n")
			}
		} else {

			if !once {
				al_writer.WriteString("// File: " + path.Base(file_name) + "\n")
				once = true
			}

			al_writer.WriteString(line + "\n")
		}
	}

	// Check for any errors during scanning
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	al_writer.WriteString("\n")
	out.WriteString(all_lines.String())
}

func main() {

	f_o := flag.String("o", "", "output file")

	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		fmt.Println("Please provide a file name.")
		return
	}

	i := args[0]
	o := *f_o

	if o == "" {
		o = strings.Replace(i, ".sol", "_flat.sol", 1)
	}

	f_out, err := os.Create(o)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer f_out.Close()

	writer := bufio.NewWriter(f_out)
	processFile("", i, writer)

	writer.WriteString("// Processed by Dirty Solidity Flattener by @AlexNa \n")
	writer.WriteString("// https://github.com/lexansoft/dirty-solidity-flattener \n")

	writer.Flush()
}
