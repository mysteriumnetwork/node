package storage

type FakeStorage struct{}

func (fs *FakeStorage) Store(issuer string, data interface{}) error  { return nil }
func (fs *FakeStorage) Delete(issuer string, data interface{}) error { return nil }
func (fs *FakeStorage) Close() error                                 { return nil }
func (fs *FakeStorage) GetAll(issuer string, data interface{}) error { return nil }
func (fs *FakeStorage) Save(data interface{}) error                  { return nil }
func (fs *FakeStorage) Update(data interface{}) error                { return nil }
func (fs *FakeStorage) GetAllSessions(data interface{}) error        { return nil }
