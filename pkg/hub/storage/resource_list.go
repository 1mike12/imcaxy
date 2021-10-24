package datahubstorage

import (
	"errors"
	"sync"
)

type resourceList struct {
	resources map[string]*threadSafeResource
	lock      sync.RWMutex
}

func newResourceList() resourceList {
	return resourceList{
		make(map[string]*threadSafeResource),
		sync.RWMutex{},
	}
}

func (list *resourceList) ReadAt(resourceID string, p []byte, off int64) (n int, err error) {
	list.lock.RLock()
	defer list.lock.RUnlock()

	resource, exists := list.resources[resourceID]
	if !exists {
		err = errUnknownResource
		return
	}

	return resource.ReadAt(p, off)
}

func (list *resourceList) Write(resourceID string, p []byte) (n int, err error) {
	list.lock.RLock()
	defer list.lock.RUnlock()

	resource, exists := list.resources[resourceID]
	if !exists {
		err = errUnknownResource
		return
	}

	return resource.Write(p)
}

func (list *resourceList) Create(resourceID string) error {
	list.lock.Lock()
	defer list.lock.Unlock()

	if _, exists := list.resources[resourceID]; exists {
		return errResourceAlreadyExists
	}

	resource := newThreadSafeResource()
	list.resources[resourceID] = &resource
	return nil
}

func (list *resourceList) Exists(resourceID string) bool {
	list.lock.RLock()
	defer list.lock.RUnlock()

	_, exists := list.resources[resourceID]
	return exists
}

func (list *resourceList) Close(resourceID string, errorToForward error) error {
	list.lock.RLock()
	defer list.lock.RUnlock()

	resource, exists := list.resources[resourceID]
	if !exists {
		return errUnknownResource
	}

	return resource.Close(errorToForward)
}

func (list *resourceList) Dispose(resourceID string) error {
	list.lock.Lock()
	defer list.lock.Unlock()

	resource, exists := list.resources[resourceID]
	if !exists {
		return errUnknownResource
	}

	resource.Close(nil)
	delete(list.resources, resourceID)
	return nil
}

var (
	errUnknownResource       = errors.New("unknown resource")
	errResourceAlreadyExists = errors.New("resource already exists")
)
