#!/bin/bash

# --- 免责声明 ---
echo "软件许可及免责声明 / Software License & Disclaimer"
echo "该软件以GPLv3许可证开源"
echo "本程序按“原样”（"AS IS"）提供，不附带任何形式的明示或暗示保证。作者不保证程序符合特定用途，亦不保证运行过程中不出现错误。"
echo "在任何情况下，作者不对因使用本程序产生的任何损害（包括数据丢失、系统崩溃、法律诉讼等）承担任何责任。作者的全部赔偿责任上限在任何情况下均不超过用户实际支付的授权费用（如有）。"
echo "用户一旦运行、调试或以任何方式使用本程序，即视为完全理解并接受上述条款。作者保留对本协议的最终解释权，并有权随时更新授权条款。"

read -p "您是否同意以上免责声明？(y/n): " agreement
if [[ "$agreement" != "y" && "$agreement" != "Y" ]]; then
    echo "用户不同意免责声明，退出安装。"
    exit 1
fi

# --- 创建工作目录 ---
WORKDIR="tanzhen"
mkdir -p "$WORKDIR"
cd "$WORKDIR" || exit

# --- 选择模式 ---
echo "请选择安装模式："
read -p "这是监控端还是被监控端？(server/client): " mode
mode=$(echo "$mode" | tr '[:upper:]' '[:lower:]')

# --- 逻辑处理：服务端 ---
if [[ "$mode" == "s" || "$mode" == "ser" || "$mode" == "server" ]]; then
    echo "正在下载服务端程序..."
    wget -O server-bin "https://node1-rn.a05777.uk:8443/dows/tanzhen/server-bin"
    chmod +x server-bin
    wget -O index.html "https://node1-rn.a05777.uk:8443/dows/tanzhen/jiankong.html"

    read -p "请输入域名 (domain): " s_domain
    read -p "请输入端口 (ports, 例如 8080)（该端口将作为客户端连接端口与Web面板端口，请搭配反向代理或者CloudFlare Tunnel使用）: " s_port
    read -p "请输入 Token（相当于密码，不要泄露给他人，记住它，后面用得到）: " s_token

    # 生成 config.json
    cat <<EOF > config.json
{
  "domain": "$s_domain",
  "port": ":$s_port",
  "allowed_tokens": [
    "$s_token"
  ],
  "is_first_run": true
}
EOF

    echo "-------------------------------------------------------"
    echo "服务端配置完成！"
    CERT_PATH="./certs/server.crt"
    if [ -f "$CERT_PATH" ]; then
        echo "请复制以下证书内容用于客户端配置："
        cat "$CERT_PATH"
    else
        echo "提示：未在 $CERT_PATH 发现证书，请确保程序运行后生成并分发给客户端。"
    fi
    echo "-------------------------------------------------------"

    # Systemctl 逻辑
    read -p "是否生成 systemctl 守护进程 (tanzhen-ser)？(y/n, 默认 n): " sys_choice
    if [[ "$sys_choice" == "y" || "$sys_choice" == "Y" ]]; then
        cat <<EOF | sudo tee /etc/systemd/system/tanzhen-ser.service
[Unit]
Description=Tanzhen Server Service
After=network.target

[Service]
Type=simple
WorkingDirectory=$(pwd)
ExecStart=$(pwd)/server-bin
Restart=always

[Install]
WantedBy=multi-user.target
EOF
        sudo systemctl daemon-reload
        sudo systemctl enable tanzhen-ser
        sudo systemctl start tanzhen-ser
        echo "tanzhen-ser 已启动并设置开机自启。"
    fi

# --- 逻辑处理：客户端 ---
elif [[ "$mode" == "c" || "$mode" == "cli" || "$mode" == "client" ]]; then
    echo "正在下载客户端程序..."
    wget -O client-bin "https://node1-rn.a05777.uk:8443/dows/tanzhen/client-bin"
    chmod +x client-bin

    echo "请填写客户端配置（需与服务端对应）："
    read -p "请输入服务端域名: " c_domain
    read -p "请输入服务端端口: " c_port
    read -p "请输入 Token  : " c_token
    read -p "请输入显示名称 (hostname): " c_name
    read -p "请输入线路信息 (network): " c_line
    read -p "请输入价格 (price): " c_price
    read -p "请输入到期日期 (date): " c_date

    cat <<EOF > config.json
{
  "server_url": "https://$c_domain:$c_port/report",
  "token": "$c_token",
  "display_name": "$c_name",
  "line_info": "$c_line",
  "price": "$c_price",
  "expiry_date": "$c_date",
  "test_port": 19198,
  "report_interval": 3,
  "ca_cert": "server.crt"
}
EOF

    # 证书写入逻辑
    while true; do
        echo "-------------------------------------------------------"
        echo "请输入服务端的证书内容。"
        echo "提示：必须以 '-----BEGIN CERTIFICATE-----' 开始，"
        echo "以 '-----END CERTIFICATE-----' 结束。"
        echo "或者输入 'yes,iknow,skipthis' 跳过此步骤。"
        echo "-------------------------------------------------------"
        
        # 使用临时文件捕获多行输入
        > server.crt
        while IFS= read -r line; do
            [[ -z "$line" ]] && break
            echo "$line" >> server.crt
        done
        
        CERT_CONTENT=$(cat server.crt)
        
        if [[ "$CERT_CONTENT" == yes,iknow,skipthis* ]]; then
            echo "已跳过证书写入。请务必在启动前手动配置 server.crt。"
            break
        elif [[ "$CERT_CONTENT" == *"-----BEGIN CERTIFICATE-----"* && "$CERT_CONTENT" == *"-----END CERTIFICATE-----"* ]]; then
            echo "证书格式校验通过。"
            break
        else
            echo "证书内容不符合预期，请重新输入！"
            > server.crt
        fi
    done

    # Systemctl 逻辑
    read -p "是否生成 systemctl 守护进程 (tanzhen-cli)？(y/n, 默认 n): " sys_choice
    if [[ "$sys_choice" == "y" || "$sys_choice" == "Y" ]]; then
        cat <<EOF | sudo tee /etc/systemd/system/tanzhen-cli.service
[Unit]
Description=Tanzhen Client Service
After=network.target

[Service]
Type=simple
WorkingDirectory=$(pwd)
ExecStart=$(pwd)/client-bin
Restart=always

[Install]
WantedBy=multi-user.target
EOF
        sudo systemctl daemon-reload
        sudo systemctl enable tanzhen-cli
        sudo systemctl start tanzhen-cli
        echo "tanzhen-cli 已尝试启动。"
        echo "如果您跳过了证书步骤，请在配置好证书后手动执行：systemctl restart tanzhen-cli"
    fi

else
    echo "输入错误，请输入 server 或 client。"
    exit 1
fi

echo "安装流程结束，如果你觉得它好用的话，别忘了给我点一颗Star哦！"
