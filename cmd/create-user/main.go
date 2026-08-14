// Command create-user 提供不经过 HTTP 的固定用户创建入口。
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

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/config"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/database"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/service"
	"golang.org/x/term"
)

// main 读取一次密码、生成 bcrypt 哈希并通过 Service/Repository 写入 MySQL。
func main() {
	username := flag.String("username", "", "要创建的用户名")
	passwordStdin := flag.Bool("password-stdin", false, "从标准输入读取一行密码（适合受控自动化）")
	flag.Parse()

	if strings.TrimSpace(*username) == "" {
		exitWithError(errors.New("必须提供 --username"))
	}
	password, err := readPassword(os.Stdin, os.Stderr, *passwordStdin)
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

	userService := service.NewUserService(repository.NewUserRepository(db))
	id, err := userService.Create(context.Background(), *username, password)
	if err != nil {
		exitWithError(err)
	}
	fmt.Fprintf(os.Stdout, "用户 %q 创建成功（id=%d）\n", strings.TrimSpace(*username), id)
}

// readPassword 默认在终端关闭回显；显式 password-stdin 时只读取一行且不记录内容。
func readPassword(input io.Reader, output io.Writer, fromStdin bool) (string, error) {
	if fromStdin {
		password, err := bufio.NewReader(input).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("读取密码: %w", err)
		}
		return strings.TrimRight(password, "\r\n"), nil
	}
	file, ok := input.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return "", errors.New("标准输入不是终端；受控自动化请显式使用 --password-stdin")
	}
	fmt.Fprint(output, "请输入密码（输入不会显示）: ")
	password, err := term.ReadPassword(int(file.Fd()))
	fmt.Fprintln(output)
	if err != nil {
		return "", fmt.Errorf("读取密码: %w", err)
	}
	return string(password), nil
}

// exitWithError 输出不含明文密码的清晰错误并退出。
func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, "创建用户失败:", err)
	os.Exit(1)
}
