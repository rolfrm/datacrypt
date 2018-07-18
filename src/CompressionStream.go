package main

import "io"
import "crypto/cipher"
import compress "compress/zlib"
import "crypto/sha256"
import "crypto/aes"
import "hash"
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

type HashWriter struct {

	hash hash.Hash
	writer io.Writer
}

func (hw HashWriter)Write(p []byte)(int, error){
	hw.hash.Write(p)
	return hw.writer.Write(p)
}

func NewHashWriter(hash hash.Hash, writer io.Writer) HashWriter{
	return HashWriter {hash: hash, writer: writer}
}

func (hw * HashWriter) Sum() []byte{
	return hw.hash.Sum(nil)
}

