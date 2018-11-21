#!/usr/bin/env bash

HOST=`hostname -s`
DOMAIN=`hostname -d`
export DEFAULT_SEARCH_LOCATIONS="classpath:/,classpath:/config/,file:./,file:./config/"
export CUSTOM_SEARCH_LOCATIONS=${DEFAULT_SEARCH_LOCATIONS},file:${BASE_DIR}/conf/
function print_servers(){
    for(( i=0; i<$NACOS_REPLICAS; i++))
    do
    echo "$NAME-$i.$DOMAIN:$NACOS_SERVER_PORT"
    done
}

function validate_env(){
echo "Validating environment"

if [ -z $NACOS_REPLICAS ]; then
    echo "NACOS_REPLICAS is a mandatory environment variable"
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



#===========================================================================================
# JVM Configuration
#===========================================================================================

if [[ "${MODE}" == "standalone" ]]; then

    JAVA_OPT="${JAVA_OPT} -Xms512m -Xmx512m -Xmn256m"
    JAVA_OPT="${JAVA_OPT} -Dnacos.standalone=true"
else

  JAVA_OPT="${JAVA_OPT} -server -Xms2g -Xmx2g -Xmn1g -XX:MetaspaceSize=128m -XX:MaxMetaspaceSize=320m"
  JAVA_OPT="${JAVA_OPT} -XX:-OmitStackTraceInFastThrow -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=${BASE_DIR}/logs/java_heapdump.hprof"
  JAVA_OPT="${JAVA_OPT} -XX:-UseLargePages"
  validate_env
	create_config
fi

JAVA_MAJOR_VERSION=$($JAVA -version 2>&1 | sed -E -n 's/.* version "([0-9]*).*$/\1/p')
if [[ "$JAVA_MAJOR_VERSION" -ge "9" ]] ; then
  JAVA_OPT="${JAVA_OPT} -Xlog:gc*:file=${BASE_DIR}/logs/nacos_gc.log:time,tags:filecount=10,filesize=102400"
else
  JAVA_OPT="${JAVA_OPT} -Xloggc:${BASE_DIR}/logs/nacos_gc.log -verbose:gc -XX:+PrintGCDetails -XX:+PrintGCDateStamps -XX:+PrintGCTimeStamps -XX:+UseGCLogFileRotation -XX:NumberOfGCLogFiles=10 -XX:GCLogFileSize=100M"
fi

if [[ "${PREFER_HOST_MODE}" == "hostname" ]]; then
    JAVA_OPT="${JAVA_OPT} -Dnacos.preferHostnameOverIp=true"
fi
JAVA_OPT="${JAVA_OPT} -Dnacos.home=${BASE_DIR}"
JAVA_OPT="${JAVA_OPT} -jar ${BASE_DIR}/target/nacos-server.jar"
JAVA_OPT="${JAVA_OPT} ${JAVA_OPT_EXT}"
JAVA_OPT="${JAVA_OPT} --spring.config.location=${CUSTOM_SEARCH_LOCATIONS}"
JAVA_OPT="${JAVA_OPT} --logging.config=${BASE_DIR}/conf/nacos-logback.xml"

echo "nacos is starting,you can check the ${BASE_DIR}/logs/start.out"
echo "$JAVA ${JAVA_OPT}" > ${BASE_DIR}/logs/start.out 2>&1 &
nohup $JAVA ${JAVA_OPT} > ${BASE_DIR}/logs/start.out 2>&1 < /dev/null
