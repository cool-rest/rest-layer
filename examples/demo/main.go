// +build go1.7

package main

import (
	"log"
	"net/http"
	"net/url"

	"golang.org/x/net/context"

	"github.com/cool-rest/alice"
	"github.com/cool-rest/cors"
	"github.com/cool-rest/rest-layer-mem"
	"github.com/cool-rest/rest-layer/resource"
	"github.com/cool-rest/rest-layer/rest"
	"github.com/cool-rest/rest-layer/schema"
	"github.com/cool-rest/xaccess"
	"github.com/cool-rest/xlog"
)

var (
	// Define a user resource schema
	user = schema.Schema{
		Fields: schema.Fields{
			"id": {
				Required: true,
				// When a field is read-only, on default values or hooks can
				// set their value. The client can't change it.
				ReadOnly: true,
				// This is a field hook called when a new user is created.
				// The schema.NewID hook is a provided hook to generate a
				// unique id when no value is provided.
				OnInit: schema.NewID,
				// The Filterable and Sortable allows usage of filter and sort
				// on this field in requests.
				Filterable: true,
				Sortable:   true,
				Validator: &schema.String{
					Regexp: "^[0-9a-v]{20}$",
				},
			},
			"created": {
				Required:   true,
				ReadOnly:   true,
				Filterable: true,
				Sortable:   true,
				OnInit:     schema.Now,
				Validator:  &schema.Time{},
			},
			"updated": {
				Required:   true,
				ReadOnly:   true,
				Filterable: true,
				Sortable:   true,
				OnInit:     schema.Now,
				// The OnUpdate hook is called when the item is edited. Here we use
				// provided Now hook which just return the current time.
				OnUpdate:  schema.Now,
				Validator: &schema.Time{},
			},
			// Define a name field as required with a string validator
			"name": {
				Required:   true,
				Filterable: true,
				Validator: &schema.String{
					MaxLen: 150,
				},
			},
		},
	}

	// Define a post resource schema
	post = schema.Schema{
		Fields: schema.Fields{
			// schema.*Field are shortcuts for common fields (identical to users' same fields)
			"id":      schema.IDField,
			"created": schema.CreatedField,
			"updated": schema.UpdatedField,
			// Define a user field which references the user owning the post.
			// See bellow, the content of this field is enforced by the fact
			// that posts is a sub-resource of users.
			"user": {
				Required:   true,
				Filterable: true,
				Validator: &schema.Reference{
					Path: "users",
				},
			},
			"published": {
				Filterable: true,
				Validator:  &schema.Bool{},
			},
			// Sub-documents are handled via a sub-schema
			"meta": {
				Schema: &schema.Schema{
					Fields: schema.Fields{
						"title": {
							Required: true,
							Validator: &schema.String{
								MaxLen: 150,
							},
						},
						"body": {
							// Dependency defines that body field can't be changed if
							// the published field is not "false".
							Dependency: schema.Q(`{"published": false}`),
							Validator: &schema.String{
								MaxLen: 100000,
							},
						},
					},
				},
			},
		},
	}
)

func main() {
	// Create a REST API resource index
	index := resource.NewIndex()

	// Add a resource on /users[/:user_id]
	users := index.Bind("users", user, mem.NewHandler(), resource.Conf{
		// We allow all REST methods
		// (rest.ReadWrite is a shortcut for []resource.Mode{resource.Create, resource.Read, resource.Update, resource.Delete, resource,List})
		AllowedModes: resource.ReadWrite,
	})

	// Bind a sub resource on /users/:user_id/posts[/:post_id]
	// and reference the user on each post using the "user" field of the posts resource.
	posts := users.Bind("posts", "user", post, mem.NewHandler(), resource.Conf{
		// Posts can only be read, created and deleted, not updated
		AllowedModes: []resource.Mode{resource.Read, resource.List, resource.Create, resource.Delete},
	})

	// Add a friendly alias to public posts
	// (equivalent to /users/:user_id/posts?filter={"published":true})
	posts.Alias("public", url.Values{"filter": []string{"{\"published\":true}"}})

	// Create API HTTP handler for the resource graph
	api, err := rest.NewHandler(index)
	if err != nil {
		log.Fatalf("Invalid API configuration: %s", err)
	}

	c := alice.New()

	// Add close notifier handler so context is cancelled when the client closes
	// the connection
	// c.Append(xhandler.CloseHandler)

	// Add timeout handler
	// c.Append(xhandler.TimeoutHandler(2 * time.Second))

	// Install a logger (see https://github.com/cool-rest/xlog)
	c.Append(xlog.NewHandler(xlog.Config{}))
	resource.LoggerLevel = resource.LogLevelDebug
	resource.Logger = func(ctx context.Context, level resource.LogLevel, msg string, fields map[string]interface{}) {
		xlog.FromContext(ctx).OutputF(xlog.Level(level), 2, msg, fields)
	}

	// Log API access
	c.Append(xaccess.NewHandler())

	// Add CORS support with passthrough option on so rest-layer can still
	// handle OPTIONS method
	c.Append(cors.New(cors.Options{OptionsPassthrough: true}).Handler)

	// Bind the API under /api/ path
	http.Handle("/api/", http.StripPrefix("/api/", c.Then(api)))

	// Serve it
	log.Print("Serving API on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
