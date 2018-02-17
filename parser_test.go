package cron

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	type args struct {
		expression string
		loc        *time.Location
		name       string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr string
	}{
		{
			name: "general expression", args: args{expression: "* * * * *", loc: time.Local},
			want: `{ name:"general expression" schedule:"* * * * *", location:"Local" }`, wantErr: "",
		},
		{
			name: "different timezone", args: args{expression: "* * * * *", loc: time.UTC},
			want: `{ name:"different timezone" schedule:"* * * * *", location:"UTC" }`, wantErr: "",
		},
		{
			name: "nil timezone", args: args{expression: "* * * * *", loc: nil},
			want: `{ name:"nil timezone" schedule:"* * * * *", location:"Local" }`, wantErr: "",
		},
		{
			name: "normal value", args: args{expression: "59 23 31 12 7", loc: time.Local},
			want: `{ name:"normal value" schedule:"59 23 31 12 7", location:"Local" }`, wantErr: "",
		},
		{
			name: "invalid field", args: args{expression: "* * * *", loc: time.UTC}, want: ``,
			wantErr: "got 4 want 5 expressions",
		},
		{
			name: "wrong minute", args: args{expression: "60 23 31 12 7", loc: time.Local}, want: ``,
			wantErr: `failed parsing 'minute' field "60": value out of range (0 - 59): 60`,
		},
		{
			name: "wrong hour", args: args{expression: "59 24 31 12 7", loc: time.Local}, want: ``,
			wantErr: `failed parsing 'hour' field "24": value out of range (0 - 23): 24`,
		},
		{
			name: "wrong day of month 0", args: args{expression: "59 23 0 12 7", loc: time.Local}, want: ``,
			wantErr: `failed parsing 'day of month' field "0": value out of range (1 - 31): 0`,
		},
		{
			name: "wrong day of month", args: args{expression: "59 23 32 12 7", loc: time.Local}, want: ``,
			wantErr: `failed parsing 'day of month' field "32": value out of range (1 - 31): 32`,
		},
		{
			name: "wrong month 0", args: args{expression: "59 23 31 0 7", loc: time.Local}, want: ``,
			wantErr: `failed parsing 'month' field "0": value out of range (1 - 12): 0`,
		},
		{
			name: "wrong month", args: args{expression: "59 23 31 13 7", loc: time.Local}, want: ``,
			wantErr: `failed parsing 'month' field "13": value out of range (1 - 12): 13`,
		},
		{
			name: "wrong dow 0", args: args{expression: "59 23 31 12 0", loc: time.Local}, want: ``,
			wantErr: `failed parsing 'day of week' field "0": value out of range (1 - 7): 0`,
		},
		{
			name: "wrong dow", args: args{expression: "59 23 31 12 8", loc: time.Local}, want: ``,
			wantErr: `failed parsing 'day of week' field "8": value out of range (1 - 7): 8`,
		},
		{
			name: "with csv", args: args{expression: "1,3,5,7,9 23 31 12 7", loc: time.Local},
			want: `{ name:"with csv" schedule:"1,3,5,7,9 23 31 12 7", location:"Local" }`, wantErr: "",
		},
		{
			name: "with csv", args: args{expression: "1,3,60 23 31 12 7", loc: time.Local}, want: ``,
			wantErr: `failed parsing 'minute' field "1,3,60": value out of range (0 - 59): 60`,
		},
		{
			name: "with step", args: args{expression: "*/2 23 31 12 7", loc: time.Local},
			want: `{ name:"with step" schedule:"0,2,4,6,8,10,12,14,16,18,20,22,24,26,28,30,32,34,36,38,40,42,44,46,48,50,52,54,56,58 23 31 12 7", location:"Local" }`, wantErr: "",
		},
		{
			name: "with step without range", args: args{expression: "30/2 23 31 12 7", loc: time.Local}, want: ``,
			wantErr: `failed parsing 'minute' field "30/2": step given without range, expression "30/2"`,
		},
		{
			name: "with step and range", args: args{expression: "10-30/3 23 31 12 7", loc: time.Local},
			want: `{ name:"with step and range" schedule:"10,13,16,19,22,25,28 23 31 12 7", location:"Local" }`, wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			arg := tt.args
			arg.name = tt.name
			e, err := Parse(arg.expression, arg.loc, arg.name)
			if (err == nil) && tt.wantErr != "" || (err != nil) && err.Error() != tt.wantErr {
				t.Errorf("Parse() error = %q, wantErr %q", err, tt.wantErr)
				return
			}
			if tt.wantErr != "" {
				// test return the correct error
				return
			}

			got := e.String()
			if got != tt.want {
				t.Errorf("got %q want %q", got, tt.want)
			}
		})
	}
}
