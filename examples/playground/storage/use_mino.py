from storage import Storage


s = Storage(
    endpoint="arcadia-minio-api.172.22.96.167.nip.io",
    access_key="admin",
    secret_key="Passw0rd!",
    secure=False
)

objects = s.list_objects(bucket_name="playground", prefix="", recursive=True)

for obj in objects:
    print(obj)
