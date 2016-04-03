//  Copyright (c) 2015 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package firestorm

import (
	"testing"
	"time"

	"github.com/blevesearch/bleve/document"
	"github.com/blevesearch/bleve/index"
	"github.com/blevesearch/bleve/index/store/gtreap"
)

var dictWaitDuration = 5 * time.Second

func TestDump(t *testing.T) {
	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewFirestorm(gtreap.Name, nil, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Fatalf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	var expectedCount uint64
	docCount, err := idx.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}

	doc := document.NewDocument("1")
	doc.AddField(document.NewTextFieldWithIndexingOptions("name", []uint64{}, []byte("test"), document.IndexField|document.StoreField))
	doc.AddField(document.NewNumericFieldWithIndexingOptions("age", []uint64{}, 35.99, document.IndexField|document.StoreField))
	dateField, err := document.NewDateTimeFieldWithIndexingOptions("unixEpoch", []uint64{}, time.Unix(0, 0), document.IndexField|document.StoreField)
	if err != nil {
		t.Error(err)
	}
	doc.AddField(dateField)
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}

	doc = document.NewDocument("2")
	doc.AddField(document.NewTextFieldWithIndexingOptions("name", []uint64{}, []byte("test2"), document.IndexField|document.StoreField))
	doc.AddField(document.NewNumericFieldWithIndexingOptions("age", []uint64{}, 35.99, document.IndexField|document.StoreField))
	dateField, err = document.NewDateTimeFieldWithIndexingOptions("unixEpoch", []uint64{}, time.Unix(0, 0), document.IndexField|document.StoreField)
	if err != nil {
		t.Error(err)
	}
	doc.AddField(dateField)
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}

	fieldsCount := 0
	fieldsRows := idx.DumpFields()
	for range fieldsRows {
		fieldsCount++
	}
	if fieldsCount != 4 { // _id field is automatic
		t.Errorf("expected 4 fields, got %d", fieldsCount)
	}

	// 1 id term
	// 1 text term
	// 16 numeric terms
	// 16 date terms
	// 3 stored fields
	expectedDocRowCount := int(1 + 1 + (2 * (64 / document.DefaultPrecisionStep)) + 3)
	docRowCount := 0
	docRows := idx.DumpDoc("1")
	for range docRows {
		docRowCount++
	}
	if docRowCount != expectedDocRowCount {
		t.Errorf("expected %d rows for document, got %d", expectedDocRowCount, docRowCount)
	}

	docRowCount = 0
	docRows = idx.DumpDoc("2")
	for range docRows {
		docRowCount++
	}
	if docRowCount != expectedDocRowCount {
		t.Errorf("expected %d rows for document, got %d", expectedDocRowCount, docRowCount)
	}

	err = idx.(*Firestorm).dictUpdater.waitTasksDone(dictWaitDuration)
	if err != nil {
		t.Fatal(err)
	}

	// 1 version
	// fieldsCount field rows
	// 2 docs * expectedDocRowCount
	// 2 text term row count (2 different text terms)
	// 16 numeric term row counts (shared for both docs, same numeric value)
	// 16 date term row counts (shared for both docs, same date value)
	//
	expectedAllRowCount := int(1 + fieldsCount + (2 * expectedDocRowCount) + 2 + int((2 * (64 / document.DefaultPrecisionStep))))
	allRowCount := 0
	allRows := idx.DumpAll()
	for range allRows {
		allRowCount++
	}
	if allRowCount != expectedAllRowCount {
		t.Errorf("expected %d rows for all, got %d", expectedAllRowCount, allRowCount)
	}
}
