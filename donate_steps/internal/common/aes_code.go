//
//

package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"

	"git.code.oa.com/gongyi/agw/log"
)

// 长度有限制，一般为16，24，32个字节
const AesKey = "gongyi-20200604-"

const AES_BLOCK_SIZE = 16

func GenerateGongYiAesKey(key_seed byte) [AES_BLOCK_SIZE]byte {
	var key [AES_BLOCK_SIZE]byte
	for i := 0; i < AES_BLOCK_SIZE; i++ {
		key[i] = key_seed + byte(i)
	}
	return key
}

// PKCS5的分组是以8为单位
// PKCS7的分组长度为1-255
func pKCS7Padding(org []byte, blockSize int) []byte {
	pad := blockSize - len(org)%blockSize
	padArr := bytes.Repeat([]byte{byte(pad)}, pad)
	return append(org, padArr...)
}

//通过AES方式解密密文
func pKCS7UnPadding(org []byte) []byte {
	//abcd4444
	l := len(org)
	pad := org[l-1]
	//org[0:4]
	return org[:l-int(pad)]
}

//AES解密
func AESDecrypt(cipherTxt string, key string) (result string) {
	defer func() {
		if err := recover(); err != nil {
			log.Error("AESDecrypt panic - cipherTxt: %v", cipherTxt)
		}
	}()
	tmp_text, _ := base64.StdEncoding.DecodeString(cipherTxt)
	tmp_key := []byte(key)
	block, _ := aes.NewCipher(tmp_key)
	blockMode := cipher.NewCBCDecrypter(block, tmp_key)
	//创建明文缓存
	org := make([]byte, len(tmp_text))
	//开始解密
	blockMode.CryptBlocks(org, tmp_text)
	//去码
	org = pKCS7UnPadding(org)
	//返回明文
	return string(org)
}

//AES加密
func AESEncrypt(org string, key string) string {
	tmp_org := []byte(org)
	tmp_key := []byte(key)
	//检验秘钥
	block, _ := aes.NewCipher(tmp_key)
	//对明文进行补码
	tmp_org = pKCS7Padding(tmp_org, block.BlockSize())
	//设置加密模式
	blockMode := cipher.NewCBCEncrypter(block, tmp_key)

	//创建密文缓冲区
	cryted := make([]byte, len(tmp_org))
	//加密
	blockMode.CryptBlocks(cryted, tmp_org)
	//返回密文
	return base64.StdEncoding.EncodeToString(cryted)
}
