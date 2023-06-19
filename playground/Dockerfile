FROM alpine:latest

COPY id_rsa.pub /root/.ssh/authorized_keys

RUN apk add --no-cache \
        bash \
        openssh \
        sudo \
    && ssh-keygen -A \
    && sed -i "s/#\?PermitRootLogin.*/PermitRootLogin yes/" /etc/ssh/sshd_config \
    && adduser -D raphaeljoie \
    && echo "root:password" | chpasswd \
    && echo "raphaeljoie:password" | chpasswd \
    && chmod 600 /root/.ssh/authorized_keys

COPY id_rsa.pub /home/raphaeljoie/.ssh/authorized_keys

RUN chmod 600 /home/raphaeljoie/.ssh/authorized_keys

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
