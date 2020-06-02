package export

import (
	"encoding/csv"
	"log"
	"os"
)

func SaveCSV(category, filename string, contents [][]string) {
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	file, err := os.Create(dir + "/" + category + "/" + filename + ".csv")
	if err != nil {
		return
	}

	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Comma = '|'
	err = writer.WriteAll(contents)
	if err != nil {
		log.Println(err)
	}

	// for _, content := range contents {
	// 	err = writer.Write(content)
	// 	if err != nil {
	// 		log.Println(err)
	// 	}
	// }

}
