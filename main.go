package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var folderStructure = map[string][]string{
	"drums": []string{
		"808",
		"kicks",
		"snares",
		"rims",
		"claps",
		"hats",
		"toms",
		"snaps",
		"percussion",
	},
	"vocals": []string{
		"female",
		"male",
	},
	"plucks": nil,
	"bells":  nil,
}

func contains(tags []string, tag string) bool {
	for _, t := range tags {
		if tag == t {
			return true
		}
	}
	return false
}

func getSortedDir(structure map[string][]string, tags []string) (string, error) {
	for k, v := range structure {
		if contains(tags, k) {
			if v == nil {
				return k, nil
			}
			for _, t := range v {
				if contains(tags, t) {
					return filepath.Join(k, t), nil
				}
			}
		}
	}
	return "", fmt.Errorf("could not sort tags: %v", tags)
}

func findSoundsDB() string {
	if runtime.GOOS == "darwin" {
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		path := filepath.Join("/System/Volumes/Data/Users", u.Username,
			"Library/Application Support/com.splice.Splice/users/default")
		entries, err := os.ReadDir(path)
		if err != nil {
			panic(err)
		}
		// Expect length to be 1
		path = filepath.Join(path, entries[0].Name(), "sounds.db")
		return path
	}
	return ""
}

type TagCount struct {
	Tag   string
	Count int
}

type TagsRank []TagCount

func (tr TagsRank) Len() int {
	return len(tr)
}

func (tr TagsRank) Less(i, j int) bool {
	return tr[i].Count > tr[j].Count
}

func (tr TagsRank) Swap(i, j int) {
	x := tr[i]
	y := tr[j]
	tr[i] = y
	tr[j] = x
}

func getTopTags(db *sql.DB) (TagsRank, error) {
	rows, err := db.Query("select tags from samples;")
	if err != nil {
		return nil, err
	}
	counts := make(map[string]*TagCount)
	for rows.Next() {
		var list string
		rows.Scan(&list)
		tags := strings.Split(list, ",")
		for _, t := range tags {
			if _, ok := counts[t]; !ok {
				counts[t] = &TagCount{
					Tag: t,
				}
			}

			counts[t].Count++
		}
	}
	var rank TagsRank
	for _, count := range counts {
		rank = append(rank, *count)
	}
	sort.Sort(rank)
	return rank, nil
}

var maxFolders = 3

func organize(db *sql.DB, rank TagsRank, targetDir string) error {
	rows, err := db.Query("select sample_type, local_path, tags from samples;")
	if err != nil {
		return err
	}

	for rows.Next() {
		var sampleType string
		var path string
		var list string
		rows.Scan(&sampleType, &path, &list)
		tags := strings.Split(list, ",")

		if len(tags) == 0 {
			log.Fatalln("no tags?", path)
		}

		if sampleType != "loop" {
			sortedPath, err := getSortedDir(folderStructure, tags)
			if err != nil {
				// fmt.Println("ERROR", err)
			} else {
				// fmt.Println(path, "=>", sortedPath)
				name := strings.TrimSuffix(filepath.Base(path), ".wav")
				mpcName := strings.ToUpper(name)
				if len(mpcName) > 16 {
					mpcName = strings.TrimPrefix(mpcName[len(mpcName)-16:], "_")
				}
				newPath := filepath.Join(targetDir, sortedPath, mpcName+".wav")

				if _, err := os.Stat(newPath); os.IsNotExist(err) {
					cmd := exec.Command("ffmpeg", "-i", path,
						"-acodec", "pcm_s16le",
						"-ar", "44100",
						newPath)
					// cmd.Stdout = os.Stdout
					// cmd.Stderr = os.Stderr
					os.MkdirAll(filepath.Dir(newPath), 0755)
					fmt.Printf("Writing %s...\n", newPath)
					err = cmd.Run()
					if err != nil {
						log.Println("error converting:", err)
					}
				}
			}
		}

		// var theseTags TagsRank
		// for _, t := range tags {
		// 	for _, r := range rank {
		// 		if r.Tag == t {
		// 			theseTags = append(theseTags, r)
		// 		}
		// 	}
		// }
		// sort.Sort(theseTags)

		// fmt.Println(theseTags)

		// newPath := ""
		// if sampleType == "loop" {
		// 	newPath = "loops"
		// } else {
		// 	newPath = "oneshots"
		// }
		// for i, t := range theseTags {
		// 	if i >= maxFolders {
		// 		break
		// 	}
		// 	newPath = filepath.Join(newPath, t.Tag)
		// }
		// newPath = filepath.Join(newPath, filepath.Base(path))
		// newPath = filepath.Join(targetDir, newPath)

		// fmt.Println("Will move", path, "to", newPath)

		// if _, err := os.Stat(newPath); os.IsNotExist(err) {
		// 	cmd := exec.Command("ffmpeg", "-i", path,
		// 		"-acodec", "pcm_s16le",
		// 		"-ar", "44100",
		// 		newPath)
		// 	// cmd.Stdout = os.Stdout
		// 	// cmd.Stderr = os.Stderr
		// 	os.MkdirAll(filepath.Dir(newPath), 0755)
		// 	fmt.Printf("Writing %s...\n", newPath)
		// 	err = cmd.Run()
		// 	if err != nil {
		// 		log.Println("error converting:", err)
		// 	}
		// }
	}

	return nil
}

func main() {
	path := findSoundsDB()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rank, err := getTopTags(db)
	if err != nil {
		log.Fatal(err)
	}

	err = organize(db, rank, "/Volumes/MPC1000DISK/Splice")
	if err != nil {
		log.Fatal(err)
	}
}
