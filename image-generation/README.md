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
change to you bucket name ${S3_BUCKET} buildmaasimage.sh line: 80

and you must provide aws-credentials, refer secret.yaml


if you do not want to upload images to s3
remove lines

```yaml
    envFrom:
    - secretRef:
        name: aws-credentials
```

from pod.yaml

remove lines
https://github.com/spectrocloud/cluster-api-provider-maas/blob/818c818131b69fe35d5637a9e7c6510f82d39f13/image-generation/buildmaasimage.sh#L61-L82


## Upload Custom Image to MAAS using maas cli
 
You must have maas cli installed on your system 

If you don't have maas cli installed refer [here](https://maas.io/docs/snap/3.0/ui/maas-cli)

The images generated inside pod can be accessed from hostpath vol
```yaml
- name: outputdir
  hostPath:
  path: /tmp/mypath
```

cd /tmp/mypath or the directory you have configured

next step assumes that you have maas-cli installed on currrent machine. 
If not first copy the image from current machine to somewhere you have maas cli installed and access to MAAS setup
```bash
scp /tmp/mypath/<image-filename> <destination-machine>

# ssh to the machine
ssh user@<destination-machine>
```

use <profile-name> profile which has access to create boot-resources or admin
```bash
# maas <= v2.9
maas <profile-name> boot-resources create name=custom/<image-display-name> architecture=amd64/generic content=<image-filename>

# maas >= v3.0
size=$(du <image-filename> | awk '{print $1;}')
sha256checksum=$(sha256sum <image-filename> | awk '{print $1;}')
maas <profile-name> boot-resources create name=custom/<image-display-name> architecture=amd64/generic sha256=$sha256checksum size=$size content@=<image-filename>
```

## spectrocloud public images 
| kubernetes Version | URL                                                                        |
|--------------------|----------------------------------------------------------------------------|
| 1.21.14            | https://maas-images-public.s3.amazonaws.com/u-2004-0-k-12114-0.tar.gz      |
| 1.22.12            | https://maas-images-public.s3.amazonaws.com/u-2004-0-k-12212-0.tar.gz      |
| 1.23.9             | https://maas-images-public.s3.amazonaws.com/u-2004-0-k-1239-0.tar.gz       |
| 1.24.3             | https://maas-images-public.s3.amazonaws.com/u-2004-0-k-1243-0.tar.gz       |
| 1.25.6             | https://maas-images-public.s3.amazonaws.com/u-2204-0-k-1256-0.tar.gz       |
| 1.26.1             | https://maas-images-public.s3.amazonaws.com/u-2204-0-k-1261-0.tar.gz       |

