package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"os"

	"github.com/jlaffaye/ftp"
	"github.com/joho/godotenv"
)

func SaveCSV(filename string, contents [][]string) error {

	_ = godotenv.Load()

	conn, err := ftp.Dial(os.Getenv("FTPAddress"))
	if err != nil {
		log.Fatal(err.Error())
		return err
	}

	err = conn.Login(os.Getenv("FTPUser"), os.Getenv("FTPPassword"))
	if err != nil {
		log.Fatal(err.Error())
		return err
	}

	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)
	defer writer.Flush()

	writer.Comma = '|'
	err = writer.WriteAll(contents)
	if err != nil {
		log.Println(err)
		return err
	}

	destinationFile := fmt.Sprintf("%s/%s", os.Getenv("FTPDirectory"), filename)
	err = conn.Stor(destinationFile, buf)
	if err != nil {
		log.Fatal(err.Error())
		return err
	}

	return nil
}
