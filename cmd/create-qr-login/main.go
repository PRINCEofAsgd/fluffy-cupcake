// Command create-qr-login 提供不经过 HTTP 的二维码登录映射创建入口。
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/config"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/database"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
)

// maxQrTextRunes 与数据库列和登录接口的最大长度保持一致。
const maxQrTextRunes = 512

// main 校验目标用户存在后写入二维码文本到用户的映射。
func main() {
	username := flag.String("username", "", "映射到的真实注册用户名")
	qrText := flag.String("qr-text", "", "二维码中的永久文本")
	qrTextStdin := flag.Bool("qr-text-stdin", false, "从标准输入读取一行二维码文本（适合长文本）")
	flag.Parse()

	if strings.TrimSpace(*username) == "" {
		exitWithError(errors.New("必须提供 --username"))
	}
	text, err := readQrText(os.Stdin, *qrText, *qrTextStdin)
	if err != nil {
		exitWithError(err)
	}

	cfg, err := config.Load()
	if err != nil {
		exitWithError(err)
	}
	if err := cfg.ValidateDatabase(); err != nil {
		exitWithError(err)
	}
	db, err := database.Open(context.Background(), cfg.Database)
	if err != nil {
		exitWithError(err)
	}
	defer db.Close()

	userRepository := repository.NewUserRepository(db)
	user, err := userRepository.GetByUsername(context.Background(), strings.TrimSpace(*username))
	if errors.Is(err, repository.ErrNotFound) {
		exitWithError(fmt.Errorf("用户 %q 不存在", strings.TrimSpace(*username)))
	}
	if err != nil {
		exitWithError(err)
	}

	qrRepository := repository.NewQrLoginRepository(db)
	if err := qrRepository.Create(context.Background(), text, user.ID); err != nil {
		exitWithError(err)
	}
	fmt.Fprintf(os.Stdout, "二维码登录映射创建成功（user=%q, id=%d）\n", user.Username, user.ID)
}

// readQrText 优先读取命令行参数，qr-text-stdin 时读取一行且不记录内容。
func readQrText(input io.Reader, fromFlag string, fromStdin bool) (string, error) {
	var text string
	if fromStdin {
		line, err := bufio.NewReader(input).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("读取二维码文本: %w", err)
		}
		text = strings.TrimRight(line, "\r\n")
	} else {
		text = fromFlag
	}
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > maxQrTextRunes {
		return "", fmt.Errorf("二维码文本必须为 1 到 %d 个字符", maxQrTextRunes)
	}
	return text, nil
}

// exitWithError 输出不含二维码文本的错误并退出。
func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, "创建二维码登录映射失败:", err)
	os.Exit(1)
}
