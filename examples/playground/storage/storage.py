from minio import Minio


class Storage(Minio):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
