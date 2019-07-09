package Test

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func GenBuffer(i int) []byte {
	token := make([]byte, i)
	if _, err := rand.Read(token); err != nil {
		log.Fatal(err)
	}
	return token
}

func GetBuffer(i int, c byte) []byte {
	token := make([]byte, i)
	for j := range token {
		token[j] = c
	}
	return token
}

func Copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func random(min, max int) int {
	return rand.Intn(max-min) + min
}

func String(n int32) string {
	buf := [11]byte{}
	pos := len(buf)
	i := int64(n)
	signed := i < 0
	if signed {
		i = -i
	}
	for {
		pos--
		buf[pos], i = '0'+byte(i%10), i/10
		if i == 0 {
			if signed {
				pos--
				buf[pos] = '-'
			}
			return string(buf[pos:])
		}
	}
}
