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
| kubernetes Version | URL                                                                        |
|--------------------|----------------------------------------------------------------------------|
| 1.17.16            | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.17.16.tar.gz |
| 1.17.17            | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.17.17.tar.gz |
| 1.18.18            | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.18.18.tar.gz |
| 1.18.19            | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.18.19.tar.gz |
| 1.19.10            | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.19.10.tar.gz |
| 1.19.11            | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.19.11.tar.gz |
| 1.19.12            | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.19.12.tar.gz |
| 1.19.13            | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.19.13.tar.gz |
| 1.20.2             | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.20.2.tar.gz  |
| 1.20.6             | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.20.6.tar.gz  |
| 1.20.7             | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.20.7.tar.gz  |
| 1.20.8             | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.20.8.tar.gz  |
| 1.20.9             | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.20.9.tar.gz  |
| 1.21.0             | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.21.0.tar.gz  |
| 1.21.1             | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.21.1.tar.gz  |
| 1.21.2             | https://maas-images-public.s3.amazonaws.com/ubuntu-1804-k8s-1.21.2.tar.gz  |

