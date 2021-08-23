package errors

// 2xx非错误
const CODE_NORMAL = 200

// K8s资源层面错误 3XX
const CODE_PARAMETER_ERROR = 301

// 组件层面错误 4XXX
const CODE_CLUSTER_FAILE = 401
const CODE_ERR_SYSTEM = 404

const CODE_ERR_UNKNOW = -1

const MSG_PARAMETER_ERROT = "parameter error %v is %v"
const MSG_NACOS_UNREACH = "nacos is nureach %s"
const MSG_NACOS_CLUSTER = ""
const MSG_POD_STATUS = ""
