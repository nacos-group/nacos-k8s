#!/usr/bin/env bash

HOST=`hostname -s`
DOMAIN=`hostname -d`
function print_servers(){
    for(( i=0; i<$NACOS_REPLICAS; i++))
    do
    echo "$NAME-$i.$DOMAIN:$NACOS_SERVER_PORT"
    done
}

function validate_env(){
echo "Validating environment"

if [ -z $NACOS_REPLICAS ]; then
    echo "ZK_REPLICAS is a mandatory environment variable"
     exit 1
fi

if [[ $HOST =~ (.*)-([0-9]+)$ ]]; then
    NAME=${BASH_REMATCH[1]}
else
    echo "Failed to extract ordinal from hostname $HOST"
    exit 1
fi
print_servers
echo "Environment validation successful"
}
function create_config(){
 rm -f $CLUSTER_CONF
 echo "Creating Nacos Server configuration"

 if [ $NACOS_REPLICAS -gt 1 ]; then
    print_servers >> $CLUSTER_CONF
 fi
 echo "Wrote Nacos Server configuration file to $NACOS_CONFIG_FILE"
}

if [ ! -f "${CLUSTER_CONF}" ]; then
  touch "${CLUSTER_CONF}"
fi
if [[ "${MODE}" == "cluster" ]]; then
  validate_env
	create_config
fi

#===========================================================================================
# JVM Configuration
#===========================================================================================
JAVA_OPT="${JAVA_OPT} -server -Xms2g -Xmx2g -Xmn1g -XX:MetaspaceSize=128m -XX:MaxMetaspaceSize=320m"
JAVA_OPT="${JAVA_OPT} -Xdebug -Xrunjdwp:transport=dt_socket,address=9555,server=y,suspend=n"
JAVA_OPT="${JAVA_OPT} -XX:+UseConcMarkSweepGC -XX:+UseCMSCompactAtFullCollection -XX:CMSInitiatingOccupancyFraction=70 -XX:+CMSParallelRemarkEnabled -XX:SoftRefLRUPolicyMSPerMB=0 -XX:+CMSClassUnloadingEnabled -XX:SurvivorRatio=8  -XX:-UseParNewGC"
JAVA_OPT="${JAVA_OPT} -verbose:gc -Xloggc:${BASE_DIR}/logs/nacos_gc.log -XX:+PrintGCDetails -XX:+PrintGCDateStamps -XX:+PrintGCApplicationStoppedTime -XX:+PrintAdaptiveSizePolicy"
JAVA_OPT="${JAVA_OPT} -Dnacos.home=${BASE_DIR}"
if [[ "${MODE}" == "standalone" ]]; then
    JAVA_OPT="${JAVA_OPT} -Dnacos.standalone=true"
fi
if [[ "${PREFER_HOST_MODE}" == "hostname" ]]; then
    JAVA_OPT="${JAVA_OPT} -Dnacos.preferHostnameOverIp=true"
fi
JAVA_OPT="${JAVA_OPT} -XX:-OmitStackTraceInFastThrow"
JAVA_OPT="${JAVA_OPT} -XX:-UseLargePages"
JAVA_OPT="${JAVA_OPT} -jar ${BASE_DIR}/target/nacos-server.jar"
JAVA_OPT="${JAVA_OPT} ${JAVA_OPT_EXT}"
JAVA_OPT="${JAVA_OPT} --spring.config.location=${CUSTOM_SEARCH_LOCATIONS}"
JAVA_OPT="${JAVA_OPT} --logging.config=${BASE_DIR}/conf/nacos-logback.xml"

echo "nacos is starting"
nohup $JAVA ${JAVA_OPT} > ${BASE_DIR}/logs/start.log 2>&1 < /dev/null