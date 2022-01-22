
package encryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
)

var (
	Crypto = new(cryptoManager)
)

type cryptoManager struct {
}

// 长度有限制，一般为16，24，32个字节
const CounponsAesKey = "d8fbe3df3c517776e27c45975185bd40"

// PKCS7Padding 填充模式
func (c *cryptoManager) PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	// Repeat()函数的功能是把切片[]byte{byte(padding)}复制padding个，然后合并成新的字节切片返回
	pretext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, pretext...)
}

// PKCS7UnPadding 填充的反向操作，删除填充字符串
func (c *cryptoManager) PKCS7UnPadding(origData []byte) ([]byte, error) {
	// 获取数据长度
	length := len(origData)
	if length == 0 {
		return nil, errors.New("加密字符串错误！")
	} else {
		// 获取填充字符串长度
		unpacking := int(origData[length-1])
		// 截取切片，删除填充字节，并且返回明文
		return origData[:(length - unpacking)], nil
	}
}

// AesEncrypt 实现加密
func (c *cryptoManager) AesEncrypt(origData []byte, key []byte) ([]byte, error) {
	// 创建加密算法实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	// 获取块的大小
	blockSize := block.BlockSize()
	// 对数据进行填充，让数据长度满足需求
	origData = c.PKCS7Padding(origData, blockSize)
	// 采用AES加密方法中CBC加密模式
	blocMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	encrypted := make([]byte, len(origData))
	// 执行加密
	blocMode.CryptBlocks(encrypted, origData)
	return encrypted, nil
}

// AesDeCrypt 实现解密
func (c *cryptoManager) AesDeCrypt(decrypted []byte, key []byte) ([]byte, error) {
	// 创建加密算法实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	// 获取块大小
	blockSize := block.BlockSize()
	// 创建加密客户端实例
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(decrypted))
	// 这个函数也可以用来解密
	blockMode.CryptBlocks(origData, decrypted)
	// 去除填充字符串
	origData, err = c.PKCS7UnPadding(origData)
	if err != nil {
		return nil, err
	}
	return origData, err
}

// EnPwdCode 加密base64
func (c *cryptoManager) EnPwdCode(pwd []byte, key []byte) (string, error) {
	result, err := c.AesEncrypt(pwd, key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(result), err
}

// DePwdCode 解密base64
func (c *cryptoManager) DePwdCode(pwd string, key []byte) ([]byte, error) {
	// 解密base64字符串
	pwdByte, err := base64.StdEncoding.DecodeString(pwd)
	if err != nil {
		return nil, err
	}
	// 执行AES解密
	return c.AesDeCrypt(pwdByte, key)
}
