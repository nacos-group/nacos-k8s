FROM centos:7.5.1804
MAINTAINER pader "huangmnlove@163.com"

ADD ./peer/on-start.sh /
ADD ./peer/on-change.sh /
ADD ./peer/plugin.sh /
COPY ./peer/peer-finder /
ADD install.sh /
RUN chmod -c 755 /install.sh /on-start.sh /on-change.sh /plugin.sh /peer-finder
ENTRYPOINT ["/install.sh"]