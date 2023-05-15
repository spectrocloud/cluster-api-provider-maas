- Set up Docker image location for `BUILDER_IMG` in Makefile.

- Run
```shell
make docker
```

- Use pod.yaml to deploy the image builder vm or machine must have kvm enabled.

- make sure you have replaced `BUILDER_IMG` in pod.yaml.

- Run
```shell
kubectl apply -f pod.yaml
```

### NOTE:
If you want to publish your images to s3 bucket replace bucket name ${S3_BUCKET} buildmaasimage.sh at [line 80](image-generation/buildmaasimage.sh#L80) with your bucket name and you must provide aws-credentials, refer secret.yaml

### NOTE:
If you do not want to upload images to s3 bucket, kindly remove below lines from [pod.yaml](image-generation/pod.yaml)
```yaml
    envFrom:
    - secretRef:
        name: aws-credentials
```
and kindly remove below lines from [buildmaasimage.sh](image-generation/buildmaasimage.sh).
```shell
  mkdir -p $HOME/.aws

  if [[ -z "${AWS_ACCESS_KEY_ID}" ]]; then
    echo "aws access key id not set exiting"
    exit 1
  fi

  if [[ -z "${AWS_SECRET_ACCESS_KEY}" ]]; then
    echo "aws access key secret not set exiting"
    exit 1
  fi

  echo "[image-bucket]
  aws_access_key_id = ${AWS_ACCESS_KEY_ID}
  aws_secret_access_key = ${AWS_SECRET_ACCESS_KEY}
  region =  us-east-1" > $HOME/.aws/credentials

  export AWS_PROFILE=image-bucket

  aws s3api put-object --acl public-read --bucket ${S3_BUCKET} --key "u-${OUTPUT_OS_VERSION}-0-k-${IMAGE_K8S_VERSION}-0.tar.gz" --body u-${OUTPUT_OS_VERSION}-0-k-${IMAGE_K8S_VERSION}-0.tar.gz

  echo 'image upload done'
```

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

## SpectroCloud public images 
| kubernetes Version | URL                                                                        |
|--------------------|----------------------------------------------------------------------------|
| 1.21.14            | https://maas-images-public.s3.amazonaws.com/u-2004-0-k-12114-0.tar.gz      |
| 1.22.12            | https://maas-images-public.s3.amazonaws.com/u-2004-0-k-12212-0.tar.gz      |
| 1.23.9             | https://maas-images-public.s3.amazonaws.com/u-2004-0-k-1239-0.tar.gz       |
| 1.24.3             | https://maas-images-public.s3.amazonaws.com/u-2004-0-k-1243-0.tar.gz       |
| 1.25.6             | https://maas-images-public.s3.amazonaws.com/u-2204-0-k-1256-0.tar.gz       |
| 1.26.1             | https://maas-images-public.s3.amazonaws.com/u-2204-0-k-1261-0.tar.gz       |

Next step assumes that you have maas-cli installed on your currrent machine.
```bash
curl https://maas-images-public.s3.amazonaws.com/u-2004-0-k-12114-0.tar.gz -o ubuntu-1804-k8s-1.21.14.tar.gz
```

Use <profile-name> profile which has access to create boot-resources or admin
```bash 
maas <profile-name> boot-resources create name=custom/<image-display-name> architecture=amd64/generic content=<image-filename>
```

