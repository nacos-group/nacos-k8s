FROM ubuntu:16.04
MAINTAINER pader "huangmnlove@163.com"

# set environment
ENV MODE="cluster" \
    PREFER_HOST_MODE="ip"\
    BASE_DIR="/home/nacos" \
    CLASSPATH=".:/home/nacos/conf:$CLASSPATH" \
    CLUSTER_CONF="/home/nacos/conf/cluster.conf" \
    JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64 \
    JAVA="/usr/lib/jvm/java-8-openjdk-amd64/bin/java"

RUN set -x \
    && apt-get update \
    && apt-get install -y openjdk-8-jre-headless wget netcat-openbsd bash\
    && apt-get autoremove -y wget \
    && apt-get install -y tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && rm -rf /var/lib/apt/lists/*


WORKDIR /home/nacos

COPY nacos $BASE_DIR



# set startup log dir
RUN chmod +x bin/docker-startup.sh \
  && chmod +x bin/peer-finder \
  && chmod +x bin/on-change.sh \
  && mkdir -p logs \
	&& cd logs \
	&& touch start.out \
	&& ln -sf /dev/stdout $BASE_DIR/logs/start.out \
	&& ln -sf /dev/stderr $BASE_DIR/logs/start.out


EXPOSE 8848
ENTRYPOINT ["bin/docker-startup.sh"]
