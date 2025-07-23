# Split your API key into parts

KEY=tceuNMR9NUUemxbVpF:4dDRYZ7QpfV9gxvCYK:By3rz8KFxYN4x4Xydngkc6gSGfY9F2PW

IFS=":" read -r CONSUMER_KEY CONSUMER_TOKEN SECRET <<< $KEY


#CONSUMER_KEY=zQjCcLzmGkeymnBh5b
#CONSUMER_TOKEN=5ay5hvtPg2u4VK3NRQ
#SECRET=GBchP6UTsNnyywF3YvtTRxFWTtnMKpHc
# Make a GET request (e.g. list machines)
curl -v \
  --header "Authorization: OAuth \
oauth_signature_method=PLAINTEXT,\
oauth_consumer_key=${CONSUMER_KEY},\
oauth_token=${CONSUMER_TOKEN},\
oauth_signature=${SECRET}" \
  http://10.11.130.11:5240/MAAS/api/2.0/subnets/

