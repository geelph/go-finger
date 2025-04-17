import socket

# 创建一个Unix域套接字
client = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)

# 连接到指定的.sock文件
socket_path = './123.sock'
try:
    client.connect(socket_path)
    print("Connected to the socket.")

    # 持续接收消息
    while True:
        data = client.recv(1024)
        if not data:
            break
        print('Received:', data.decode())


finally:
    client.close()
