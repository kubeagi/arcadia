## How to develop

**There can only be one Query structure in all schemas, and Mutation can have one or none.**
**Query and Mutation define the entry point to the service. The fields, functions, defined below these two structures are what the front-end uses to query.**


Now that there is a `datasource` definition in the code, we can add new structures and related functions as follows.

### Add schema

go to `graph` dir.

```shell
cat << EOF > x.graphqls
type X {
  name: String
}
EOF
```

### Add a query function

The `graph/datasource.graphqls` file defines the Query structure, so we need to edit this file to add a function.

```shell
type Query {
    ds(input: QueryDatasource!): [Datasource!]
    findX(input: String!): String!
}
```

### generate resolver

in the root dir of the project:

```shell
make gql-gen
```

Looking at `datasource.resolver.go`, I can see the addition of the function we defined:

```go
func (r *queryResolver) FindX(ctx context.Context, input string) (string, error) {
   panic(fmt.Errorf("not implemented: FindX - findX"))
}
```

The file `model/model_gen.go` has a new structure x that we defined.

All we have to do is just implement the `FindX` function. And the content of the function is up to you to play with.
