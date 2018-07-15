package main

import "io"
import "crypto/cipher"
import compress "compress/zlib"
import "crypto/sha256"
import "crypto/aes"

func CompressionWriter(outwriter io.Writer, key string) io.WriteCloser{
	hsh := sha256.New()
	io.WriteString(hsh, key)
	cryptkey := hsh.Sum(nil)
	
	block, err := aes.NewCipher(cryptkey)
	if err != nil {
		panic(err)
	}
	var iv [aes.BlockSize]byte
	stream := cipher.NewOFB(block, iv[:])
	writer := &cipher.StreamWriter{S: stream, W: outwriter}
	compressed,_ := compress.NewWriterLevel(writer, compress.BestSpeed);
	return compressed;
}

func CompressionReader(inreader io.Reader, key string) io.ReadCloser {
	hsh := sha256.New()
	io.WriteString(hsh, key)
	cryptkey := hsh.Sum(nil)
	
	block, err := aes.NewCipher(cryptkey)
	if err != nil {
		panic(err)
	}
	var iv [aes.BlockSize]byte
	stream := cipher.NewOFB(block, iv[:])
	reader := &cipher.StreamReader{S: stream, R: inreader}
	decompressed,_ := compress.NewReader(reader)
	return decompressed

}
