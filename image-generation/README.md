Set up docker image location

BUILDER_IMG in Makefile

run
```
make docker
```

use pod.yaml to deploy the image builder
vm or machine must have kvm enabled

make sure you have replaced 

BUILDER_IMG in pod.yaml

run
```shell
kubectl apply -f pod.yaml
```

if you want to publish you images to s3 bucket 
change to you bucket name buildmaasimage.sh line: 80

and you must provide aws-credentials, refer secret.yaml


if you do not want to upload images to s3
remove lines

```yaml
    envFrom:
    - secretRef:
        name: aws-credentials
```

from pod.yaml


## spectrocloud public images 
| kubernetes Version | URL |
|--------------------|-----|
| 1.19.11            | TBD |

