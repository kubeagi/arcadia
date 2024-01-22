# make a setup.py for ragacli package

from setuptools import find_packages, setup

with open("README.md", "r", encoding="utf-8") as f:
    long_description = f.read()

setup(
    name="ragas_once",
    version="0.0.1",
    author="Kielo",
    author_email="lanture1064@gmail.com",
    description="A one-step cli tool for RAGAS",
    url="https://github.com/kubeagi/arcadia/pypi/ragas_once",
    long_description=long_description,
    long_description_content_type="text/markdown",
    packages=find_packages(),
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
    python_requires=">=3.8",
    install_requires=[
        "ragas",
        "langchain==0.0.354",
    ],
    entry_points={"console_scripts": ["ro = ragas_once.cli:main"]},
)
