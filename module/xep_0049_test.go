/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"testing"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0049_Matching(t *testing.T) {
	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	x := NewXEPPrivateStorage(nil)
	defer x.Done()

	require.Equal(t, []string{}, x.AssociatedNamespaces())

	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())
	require.False(t, x.MatchesIQ(iq))

	iq.AppendElement(xml.NewElementNamespace("query", privateStorageNamespace))
	require.True(t, x.MatchesIQ(iq))
}

func TestXEP0049_InvalidIQ(t *testing.T) {
	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	stm := c2s.NewMockStream("abcd", j)
	stm.SetUsername("romeo")

	x := NewXEPPrivateStorage(stm)
	defer x.Done()

	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())
	q := xml.NewElementNamespace("query", privateStorageNamespace)
	iq.AppendElement(q)

	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xml.ResultType)
	stm.SetUsername("ortuman")
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xml.GetType)
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrNotAcceptable.Error(), elem.Error().Elements().All()[0].Name())

	exodus := xml.NewElementNamespace("exodus", "exodus:ns")
	exodus.AppendElement(xml.NewElementName("exodus2"))
	q.AppendElement(exodus)
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrNotAcceptable.Error(), elem.Error().Elements().All()[0].Name())

	exodus.ClearElements()
	exodus.SetNamespace("jabber:client")
	iq.SetType(xml.SetType)
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrNotAcceptable.Error(), elem.Error().Elements().All()[0].Name())

	exodus.SetNamespace("")
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())
}

func TestXEP0049_SetAndGetPrivate(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})

	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	stm := c2s.NewMockStream("abcd", j)
	stm.SetUsername("ortuman")

	x := NewXEPPrivateStorage(stm)
	defer x.Done()

	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())
	q := xml.NewElementNamespace("query", privateStorageNamespace)
	iq.AppendElement(q)

	exodus1 := xml.NewElementNamespace("exodus1", "exodus:ns")
	exodus2 := xml.NewElementNamespace("exodus2", "exodus:ns")
	q.AppendElement(exodus1)
	q.AppendElement(exodus2)

	// set error
	storage.ActivateMockedError()
	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	storage.DeactivateMockedError()

	// set success
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())

	// get error
	q.RemoveElements("exodus2")
	iq.SetType(xml.GetType)

	storage.ActivateMockedError()
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	storage.DeactivateMockedError()

	// get success
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())

	q2 := elem.Elements().ChildNamespace("query", privateStorageNamespace)
	require.Equal(t, 2, q2.Elements().Count())
	require.Equal(t, "exodus:ns", q2.Elements().All()[0].Namespace())

	// get non existing
	exodus1.SetNamespace("exodus:ns:2")
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())
	q3 := elem.Elements().ChildNamespace("query", privateStorageNamespace)
	require.Equal(t, 1, q3.Elements().Count())
	require.Equal(t, "exodus:ns:2", q3.Elements().All()[0].Namespace())
}
