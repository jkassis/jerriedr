package schema

import (
	"fmt"
	"strings"
	"time"
)

type ArchiveFile struct {
	Archive *Archive
	Name    string
	Time    time.Time
}

func (af *ArchiveFile) TimestampParseFromName() error {
	if !strings.HasSuffix(af.Name, ".bak") {
		return fmt.Errorf("%s does not appear to be a .bak file", af.Name)
	}
	timestampString := af.Name[:len(af.Name)-4]
	timestamp, err := time.Parse(time.RFC3339, timestampString)
	if err != nil {
		return fmt.Errorf("%s does not appear to have an RFC3339 compliant name", af.Name)
	}
	af.Time = timestamp
	return nil
}

func (af *ArchiveFile) FilterIsOK(tf *TimeFilter) bool {
	return tf.isOK(af.Time)
}