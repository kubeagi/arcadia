# Data Processing 

## Current Version Main Features

Data Processing is used for data processing through MinIO, databases, Web APIs, etc. The data types handled include:
- txt
- json  
- doc
- html
- excel
- csv
- pdf
- markdown
- ppt

### Current Text Type Processing  

The data processing process includes: cleaning abnormal data, filtering, de-duplication, and anonymization.

## Design

![Design](../assets/data_process.drawio.png)

## Local Development
### Software Requirements

Before setting up the local data-process environment, please make sure the following software is installed:

- Python 3.10.x

### Environment Setup

Install the Python dependencies in the requirements.txt file

### Running

Run the server.py file in the src directory

# isort
isort is a tool for sorting imports alphabetically within your Python code. It helps maintain a consistent and clean import order. 

## install
```shell
pip install isort
```

## isort a file
```shell
isort src/server.py
```

## isort a directory
```shell
isort .
```

