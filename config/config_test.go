package config_test

import (
	"github.com/cloudfoundry-incubator/varz-firehose-nozzle/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
)

var _ = Describe("Config", func() {
	var configPath string
	var configString string

	BeforeEach(func() {
		configString = `
{
  "Index": 1,
  "Job": "varz-nozzle",
  "Deployment": "my-cf",
  "UAAUrl": "uaa.my-cf.com",
  "UAAUser": "uaa",
  "UAAPass": "password",
  "NatsType": "VarzNozzle",
  "InsecureSSLSkipVerify": true,
  "TrafficControllerURL": "doppler.my-cf.com",
  "FirehoseSubscriptionID": "firehose-a",
  "Syslog": "syslog.drain",
  "VarzPort": 8080,
  "VarzUser": "varz",
  "VarzPass": "password",
  "NatsHosts": [ "10.0.0.1", "10.0.0.2" ],
  "NatsPort": 4222,
  "NatsUser": "nats",
  "NatsPass": "password",
  "CollectorRegistrarIntervalMilliseconds": 1000
}
`
	})

	JustBeforeEach(func() {
		configPath = "/tmp/config.json"
		data := []byte(configString)
		err := ioutil.WriteFile(configPath, data, 0644)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.Remove(configPath)
	})

	It("parses the config", func() {
		c, err := config.ParseConfig(configPath)
		Expect(err).NotTo(HaveOccurred())

		Expect(c.Index).To(Equal(uint(1)))
		Expect(c.Job).To(Equal("varz-nozzle"))
		Expect(c.Deployment).To(Equal("my-cf"))
		Expect(c.UAAURL).To(Equal("uaa.my-cf.com"))
		Expect(c.UAAUser).To(Equal("uaa"))
		Expect(c.UAAPass).To(Equal("password"))
		Expect(c.NatsType).To(Equal("VarzNozzle"))
		Expect(c.InsecureSSLSkipVerify).To(BeTrue())
		Expect(c.TrafficControllerURL).To(Equal("doppler.my-cf.com"))
		Expect(c.FirehoseSubscriptionID).To(Equal("firehose-a"))
	})

	It("embeds a cfcomponent config", func() {
		c, err := config.ParseConfig(configPath)
		Expect(err).NotTo(HaveOccurred())

		Expect(c.Syslog).To(Equal("syslog.drain"))
		Expect(c.VarzPort).To(Equal(uint16(8080)))
		Expect(c.VarzUser).To(Equal("varz"))
		Expect(c.VarzPass).To(Equal("password"))
		Expect(c.NatsHosts).To(Equal([]string{"10.0.0.1", "10.0.0.2"}))
		Expect(c.NatsPort).To(Equal(4222))
		Expect(c.NatsUser).To(Equal("nats"))
		Expect(c.NatsPass).To(Equal("password"))
		Expect(c.CollectorRegistrarIntervalMilliseconds).To(Equal(1000))
	})

	Context("when the deployment field is missing", func() {
		BeforeEach(func() {
			configString = `
{
  "Index": 1,
  "Job": "varz-nozzle",
  "UAAUrl": "uaa.my-cf.com",
  "UAAUser": "uaa",
  "UAAPass": "password",
  "NatsType": "VarzNozzle",
  "InsecureSSLSkipVerify": true,
  "TrafficControllerURL": "doppler.my-cf.com",
  "FirehoseSubscriptionID": "firehose-a",
  "Syslog": "syslog.drain",
  "VarzPort": 8080,
  "VarzUser": "varz",
  "VarzPass": "password",
  "NatsHosts": [ "10.0.0.1", "10.0.0.2" ],
  "NatsPort": 4222,
  "NatsUser": "nats",
  "NatsPass": "password",
  "CollectorRegistrarIntervalMilliseconds": 1000
}
`
		})

		It("returns an error", func() {
			_, err := config.ParseConfig(configPath)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("missing field: deployment"))
		})
	})

	Context("when the job field is missing", func() {
		BeforeEach(func() {
			configString = `
{
  "Index": 1,
  "Deployment": "my-cf",
  "UAAUrl": "uaa.my-cf.com",
  "UAAUser": "uaa",
  "UAAPass": "password",
  "NatsType": "VarzNozzle",
  "InsecureSSLSkipVerify": true,
  "TrafficControllerURL": "doppler.my-cf.com",
  "FirehoseSubscriptionID": "firehose-a",
  "Syslog": "syslog.drain",
  "VarzPort": 8080,
  "VarzUser": "varz",
  "VarzPass": "password",
  "NatsHosts": [ "10.0.0.1", "10.0.0.2" ],
  "NatsPort": 4222,
  "NatsUser": "nats",
  "NatsPass": "password",
  "CollectorRegistrarIntervalMilliseconds": 1000
}
`
		})

		It("returns an error", func() {
			_, err := config.ParseConfig(configPath)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("missing field: job"))
		})
	})
})
