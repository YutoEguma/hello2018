package sshTC

import (
	"golang.org/x/net/context"
	"net"
	"google.golang.org/grpc/credentials"
	"fmt"
	mrand "math/rand"
	"crypto/sha256"
	"strings"
	"errors"
	"log"
)

type sshTC struct {
	info *credentials.ProtocolInfo
	publicKeyPath string
	privateKeyPath string
}

const rs3Letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (tc *sshTC) randString() string {
	b := make([]byte, 10)
	for i := range b {
		b[i] = rs3Letters[int(mrand.Int63()%int64(len(rs3Letters)))]
	}
	return string(b)
}

func (tc *sshTC) ClientHandshake(ctx context.Context, addr string, rawConn net.Conn) (_ net.Conn, _ credentials.AuthInfo, err error) {
	// サーバーから暗号化された乱数を受信
	buf := make([]byte, 2014)
	n, err := rawConn.Read(buf)
	if err != nil {
		log.Printf("[ERROR] Read error: %s\n", err)
		return nil, nil, err
	}

	// 復号
	key, err := tc.readPrivateKey(tc.privateKeyPath)

	decrypted, err := tc.Decrypt(string(buf[:n]), key)
	if err != nil {
		log.Printf("[ERROR] Failed to decrypt: %s\n", err)
		return nil, nil, err
	}

	// 復号結果からハッシュ値を生成し、サーバーに送信
	h := sha256.Sum256([]byte(decrypted))
	rawConn.Write([]byte(fmt.Sprintf("%x\n", h)))

	// 認証結果をサーバーから受信
	r := make([]byte, 64)
	n, err = rawConn.Read(r)
	if err != nil {
		log.Printf("[ERROR] Read error: %s\n", err)
		return nil, nil, err
	}
	r = r[:n]
	if string(r) != "ok" {
		log.Println("[ERROR] Failed to authenticate")
		return nil, nil, errors.New("Failed to authenticate")
	}

	return rawConn, nil, err
}

func (tc *sshTC) ServerHandshake(rawConn net.Conn) (_ net.Conn, _ credentials.AuthInfo, err error) {
	// 乱数を生成する
	s := tc.randString()

	// 乱数のハッシュ値を生成
	h := sha256.Sum256([]byte(s))

	// 乱数を暗号化してクライアントに送信
	encrypted, err := tc.Encrypt(s, tc.publicKeyPath)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Failed to encrypt: %s\n", err))
	}
	//fmt.Printf("encrypted: %s\n", encrypted)
	rawConn.Write([]byte(encrypted))

	// クライアントからハッシュ値を受け取る
	buf := make([]byte, 2014)
	n, err := rawConn.Read(buf)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Read error: %s\n", err))
	}

	// 事前に生成したハッシュ値とクライアントから受け取ったハッシュ値を比較する
	// 一致していれば正しいキーペアを使用していることがわかる
	if strings.TrimRight(string(buf[:n]), "\n") == fmt.Sprintf("%x", h) {
		rawConn.Write([]byte("ok"))
		log.Println("[INFO] Authenticate Success!!!")
	} else {
		rawConn.Write([]byte("ng"))
		log.Println("[ERROR] Authenticate Failed...")
		return nil, nil, errors.New(fmt.Sprintf("Failed to authenticate: invalid key"))
	}

	return rawConn, nil, err
}

func (tc *sshTC) Info() credentials.ProtocolInfo {
	return *tc.info
}

func (tc *sshTC) Clone() credentials.TransportCredentials {
	info := *tc.info
	return &sshTC{
		info: &info,
	}
}

func (tc *sshTC) OverrideServerName(serverNameOverride string) error {
	return nil
}

func NewServerCreds(path string) credentials.TransportCredentials {
	return &sshTC{
		info: &credentials.ProtocolInfo{
			SecurityProtocol: "ssh",
			SecurityVersion: "1.0",
		},
		publicKeyPath: path,
	}
}

func NewClientCreds(path string) credentials.TransportCredentials {
	return &sshTC{
		info: &credentials.ProtocolInfo{
			SecurityProtocol: "ssh",
			SecurityVersion:  "1.0",
		},
		privateKeyPath: path,
	}
}