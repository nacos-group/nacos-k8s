# Install Nacos Server On Openshift

## Version

with okd:3.11

## Install

```bash
oc process -f nacos-template.yaml -p STORAGECLASS={your_storage-class} -p NAMESPACE={the namepsace you want install} |  oc create -f -
```

## Uninstall



```bash
oc process -f nacos-template.yaml -p STORAGECLASS={your_storage-class} -p NAMESPACE={the namepsace you want uninstall} |  oc delete -f -
```