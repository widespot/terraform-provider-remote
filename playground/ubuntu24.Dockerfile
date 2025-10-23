FROM ubuntu:24.04

# Copy SSH public key for root user
COPY id_rsa.pub /root/.ssh/authorized_keys

# Install required packages and configure SSH
RUN apt-get update && apt-get install -y \
        bash \
        openssh-server \
        sudo \
#        rsyslog \
    && mkdir -p /var/run/sshd \
    && ssh-keygen -A \
    && sed -i 's/^#\?MaxSessions.*/MaxSessions 50/' /etc/ssh/sshd_config \
    && sed -i 's/^#\?LogLevel.*/LogLevel DEBUG3/' /etc/ssh/sshd_config \
    && sed -i 's/#\?PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config \
    && useradd -m -s /bin/bash raphaeljoie \
    && echo 'root:password' | chpasswd \
    && echo 'raphaeljoie:password' | chpasswd \
    && mkdir -p /home/raphaeljoie/.ssh \
    && chmod 700 /home/raphaeljoie/.ssh \
    && chmod 600 /root/.ssh/authorized_keys

# Copy SSH key for non-root user
COPY id_rsa.pub /home/raphaeljoie/.ssh/authorized_keys

RUN chmod 600 /home/raphaeljoie/.ssh/authorized_keys \
    && chown -R raphaeljoie:raphaeljoie /home/raphaeljoie/.ssh

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D", "-e"]
