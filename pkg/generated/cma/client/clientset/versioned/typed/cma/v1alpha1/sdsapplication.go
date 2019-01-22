/*
Copyright 2019 Samsung SDS Cloud Native Computing Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/samsung-cnct/cma-operator/pkg/apis/cma/v1alpha1"
	scheme "github.com/samsung-cnct/cma-operator/pkg/generated/cma/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// SDSApplicationsGetter has a method to return a SDSApplicationInterface.
// A group's client should implement this interface.
type SDSApplicationsGetter interface {
	SDSApplications(namespace string) SDSApplicationInterface
}

// SDSApplicationInterface has methods to work with SDSApplication resources.
type SDSApplicationInterface interface {
	Create(*v1alpha1.SDSApplication) (*v1alpha1.SDSApplication, error)
	Update(*v1alpha1.SDSApplication) (*v1alpha1.SDSApplication, error)
	UpdateStatus(*v1alpha1.SDSApplication) (*v1alpha1.SDSApplication, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.SDSApplication, error)
	List(opts v1.ListOptions) (*v1alpha1.SDSApplicationList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.SDSApplication, err error)
	SDSApplicationExpansion
}

// sDSApplications implements SDSApplicationInterface
type sDSApplications struct {
	client rest.Interface
	ns     string
}

// newSDSApplications returns a SDSApplications
func newSDSApplications(c *CmaV1alpha1Client, namespace string) *sDSApplications {
	return &sDSApplications{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the sDSApplication, and returns the corresponding sDSApplication object, and an error if there is any.
func (c *sDSApplications) Get(name string, options v1.GetOptions) (result *v1alpha1.SDSApplication, err error) {
	result = &v1alpha1.SDSApplication{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("sdsapplications").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of SDSApplications that match those selectors.
func (c *sDSApplications) List(opts v1.ListOptions) (result *v1alpha1.SDSApplicationList, err error) {
	result = &v1alpha1.SDSApplicationList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("sdsapplications").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested sDSApplications.
func (c *sDSApplications) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("sdsapplications").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a sDSApplication and creates it.  Returns the server's representation of the sDSApplication, and an error, if there is any.
func (c *sDSApplications) Create(sDSApplication *v1alpha1.SDSApplication) (result *v1alpha1.SDSApplication, err error) {
	result = &v1alpha1.SDSApplication{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("sdsapplications").
		Body(sDSApplication).
		Do().
		Into(result)
	return
}

// Update takes the representation of a sDSApplication and updates it. Returns the server's representation of the sDSApplication, and an error, if there is any.
func (c *sDSApplications) Update(sDSApplication *v1alpha1.SDSApplication) (result *v1alpha1.SDSApplication, err error) {
	result = &v1alpha1.SDSApplication{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("sdsapplications").
		Name(sDSApplication.Name).
		Body(sDSApplication).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *sDSApplications) UpdateStatus(sDSApplication *v1alpha1.SDSApplication) (result *v1alpha1.SDSApplication, err error) {
	result = &v1alpha1.SDSApplication{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("sdsapplications").
		Name(sDSApplication.Name).
		SubResource("status").
		Body(sDSApplication).
		Do().
		Into(result)
	return
}

// Delete takes name of the sDSApplication and deletes it. Returns an error if one occurs.
func (c *sDSApplications) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("sdsapplications").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *sDSApplications) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("sdsapplications").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched sDSApplication.
func (c *sDSApplications) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.SDSApplication, err error) {
	result = &v1alpha1.SDSApplication{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("sdsapplications").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
