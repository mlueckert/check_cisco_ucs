// 	file: check_cisco_ucs.go
// 	Version 0.9 (11.06.2019)
//
// check_cisco_ucs is a Nagios plugin made by Herwig Grimm (herwig.grimm at aon.at)
// to monitor Cisco UCS rack and blade center hardware.
//
// I have used the Google Go progamming language because of no need to install
// any libraries.
//
// The plugin uses the Cisco UCS XML API via HTTPS to do a wide variety of checks.
//
//
// This nagios plugin is free software, and comes with ABSOLUTELY NO WARRANTY.
// It may be used, redistributed and/or modified under the terms of the GNU
// General Public Licence (see http://www.fsf.org/licensing/licenses/gpl.txt).
//
// tested with:
// 	1. UCSC-C240-M3S server and CIMC firmware version 1.5(1f).24
// 	2. Cisco UCS Manager version 2.1(1e) and UCSB-B22-M3 blade center
//  3. Cisco UCS Manager version 2.2(1b) and UCSB-B200-M3
//  4. UCSC-C220-M4S server and CIMC firmware version 2.0(4c).36
//  5. UCS C240 M4S and CIMC firmware version 3.0(3a)
//  6. Cisco UCS Manager version 3.2(3g)
//
// see also:
//  	Cisco UCS Rack-Mount Servers Cisco IMC XML API Programmer's Guide, Release 3.1
// 		https://www.cisco.com/c/en/us/td/docs/unified_computing/ucs/c/sw/api/3_1/b_Cisco_IMC_api_31.html
//
//changelog:
// 	Version 0.1 (11.06.2013) initial release
//
//	Version 0.2 (26.06.2013)
//		usage text debug flag added,
//		write errors to stdout instead of stderr,
//		flag -E to show environment variables added
//		flag -V to print plugin version added
//
//	Version 0.3 (24.04.2014)
//		flag -z *OK if zero instances* added
//
//	Version 0.4 (24.02.2015)
//		flag -F display only faults in output, newlines between objects in output line
//
//	Version 0.5 (19.05.2015)
//		fix for: "remote error: handshake failure"
//		see: TLSClientConfig ... MaxVersion: tls.VersionTLS11, ...
//
//	Version 0.6 (20.07.2017)
//		fix for: " Post https://<ipaddr>/nuova/: read tcp <ipaddr>:443: connection reset by peer"
//		see: TLSClientConfig ... MaxVersion: tls.VersionTLS12, ...
//
// 		flag -M *max TLS Version* added
//
//		fix for: "HTTP 403 Forbidden error"
//		error in URL path: no backslash after *nuova*
//		see code line: url := "https://" + ipAddr + "/nuova"
//		old: .../nuova/ new: .../nuova
//
//	Version 0.7 (19.11.2018)
//		flag -f property filter added. But right now there is no support of composite filters.
//			property filter -f <type>:<property>:<value>, examples: -f wcard:dn:^sys/chassis-[1-3].*
//			works only with query type class (-t class)
// 			see also: Cisco UCS Manager XML API Programmer's Guide
//			Chapter "Using Filters" > "Property Filters"
//			https://www.cisco.com/c/en/us/td/docs/unified_computing/ucs/sw/api/b_ucs_api_book/b_ucs_api_book_chapter_01.html?bookSearch=true#d2466e1249a1635
//
//  Version 0.8 (14.05.2019)
//		jhedlund: Changed InFilter to a pointer so that it is ommitted if empty
//
//  Version 0.9 (11.06.2019)
//		repair of flag -z function *OK if zero instances* if combined with flag -f
//
// todo:
// 	1. better error handling
// 	2. add performance data support
// 	3. command line flag to influence TLS cert verification
//  4. add warning and critical thresholds
//  5. add "composite filters" to "property filters"
//
// flags:
// 	-H <ip_addr>		CIMC IP address or Cisco UCS Manager IP address"
// 	-t <query_type>		query type 'dn' or 'class'"
// 	-q <dn_or_class>	XML API object class name, examples: storageVirtualDrive or storageLocalDisk or storageControllerProps
// 						Distinguished Name (DN) name, examples: "sys/rack-unit-1"
// 	-o <object>			if XML API object class name, examples: storageVirtualDrive or storageLocalDisk or storageControllerProp
// 	-s <hierarchical>	true or false. If true, the inHierarchical argument returns all child objects
// 	-a <attributes>		space separated list of XML attributes for display in nagios output and match against *expect* string
// 	-e <expect_string>	expect string, ok if this is found, examples: "Optimal" or "Good" or "Optimal|Good"
// 	-u <username>		XML API username
// 	-p <password>		XML API password
//	-d <level>			print debug, level: 1 errors only, 2 warnings and 3 informational messages
//	-E 			print environment variables for debug purpose
//	-V			print plugin version
//	-z			true or false. if set to true the check will return OK status if zero instances where found. Default is false.
//  -F			display only faults in output
//  -M 			max TLS Version, default: v1.1"
//  -f			property filter <type>:<property>:<value>, works only with query type class (-t class), examples: wcard:dn:^sys/chassis-[1-3].*
//
// usage examples:
//
// 	Cisco UCS rack server via CIMC:
//
// 	$ ./check_cisco_ucs -H 10.18.4.7 -t class -q storageVirtualDrive -a "raidLevel vdStatus health" -e Optimal -u admin -p pls_change
// 	OK - Cisco UCS storageVirtualDrive (raidLevel,vdStatus,health) RAID 10,Optimal,Good (1 of 1 ok)
//
// 	$ ./check_cisco_ucs -H 10.18.4.7 -t class -q storageLocalDisk -a "id pdStatus driveSerialNumber" -e Online -u admin -p pls_change
// 	OK - Cisco UCS storageLocalDisk (id,pdStatus,driveSerialNumber) 1,Online,6XP4QRVQ 2,Online,6XP4QS1G 3,Online,6XP4RT6A 4,Online,6XP4RT8V (4 of 4 ok)
//
// 	$ ./check_cisco_ucs -H 10.18.64.10 -t class -q equipmentPsu -a "id model operState serial" -e operable -u admin -p pls_change
// 	CRIT - Cisco UCS equipmentPsu (id,model,operState,serial) 1,UCS-PSU-6248UP-AC,operable,POG164371G8 2,UCS-PSU-6248UP-AC,operable,POG1643721D 1,UCS-PSU-6248UP-AC,operable,POG164371C5 2,UCS-PSU-6248UP-AC,operable,POG1643721S 1,UCSB-PSU-2500ACPL,operable,AZS16210FFA 2,UCSB-PSU-2500ACPL,operable,AZS16210FH3 3,UCSB-PSU-2500ACPL,operable,AZS16210FH2 4,,removed (7 of 8 ok)
//
// 	$ ./check_cisco_ucs -H 10.18.4.7 -t dn -q sys/rack-unit-1/indicator-led-4 -o equipmentIndicatorLed -a "id color name" -e green -u admin -p pls_change
// 	OK - Cisco UCS sys/rack-unit-1/indicator-led-4 (id,color,name) 4,green,LED_FAN_STATUS (1 of 1 ok)
//
//  $ ./check_cisco_ucs -H 10.1.1.235 -t dn -q sys/rack-unit-1/indicator-led-4 -a "id color name" -e "green" -u admin -p pls_change -o equipmentIndicatorLed -M 1.2
//  OK - Cisco UCS sys/rack-unit-1/indicator-led-4 (id,color,name)
//  4,green,LED_HLTH_STATUS (1 of 1 ok)
//
// 	Cisco UCS Manager:
//
// 	$ ./check_cisco_ucs -H 10.18.64.10 -t class -q equipmentPsu -a "id model operState serial" -e operable -u admin -p pls_change
// 	CRIT - Cisco UCS equipmentPsu (id,model,operState,serial) 1,UCS-PSU-6248UP-AC,operable,POG164371G8 2,UCS-PSU-6248UP-AC,operable,POG1643721D 1,UCS-PSU-6248UP-AC,operable,POG164371C5 2,UCS-PSU-6248UP-AC,operable,POG1643721S 1,UCSB-PSU-2500ACPL,operable,AZS16210FFA 2,UCSB-PSU-2500ACPL,operable,AZS16210FH3 3,UCSB-PSU-2500ACPL,operable,AZS16210FH2 4,,removed (7 of 8 ok)
//
// 	$ ./check_cisco_ucs -H 10.18.64.10 -t dn -q sys/switch-B/slot-1/switch-ether/port-1 -o etherPIo -a operState -e up -u admin -p pls_change
// 	OK - Cisco UCS sys/switch-B/slot-1/switch-ether/port-1 (operState) up (1 of 1 ok)
//
//  $ ./check_cisco_ucs -H 10.18.64.10 -t class -q faultInst -a "code severity ack" -e "cleared,no|cleared,yes|info,no|info,yes|warning,no|warning,yes|yes|^$" -z true -u admin -p pls_change
//  OK - Cisco UCS faultInst (code,severity,ack) (0 of 0 ok)
//
//  $ ./check_cisco_ucs -H 172.18.37.164 -t class -q faultInst -a "code rn descr" -z -F -u sysu_git_ucsmon -p pls_change -s true -f "wcard:descr:^Log capacity.*"
//  OK - Cisco UCS faultInst (code,rn,descr)
//  F0461,,Log capacity on Management Controller on server 1/4 is very-low
//  F0461,,Log capacity on Management Controller on server 1/1 is very-low (0 of 2 ok)
//
//  $ ./check_cisco_ucs -H 172.18.37.164 -t class -q equipmentPsuStats -a "dn outputPower ambientTempAvg timeCollected" -z -F -u sysu_git_ucsmon -p pls_change -s true -f gt:ambientTempAvg:24
//  OK - Cisco UCS equipmentPsuStats (dn,outputPower,ambientTempAvg,timeCollected)
//  sys/chassis-3/psu-3/stats,374.696991,24.307692,2018-11-20T07:57:19.396
//  sys/chassis-2/psu-4/stats,300.200012,25.666668,2018-11-20T07:57:42.627 (0 of 2 ok)
//
package main

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
)

const (
	maxNumAttrib = 10
	version      = "0.7"
)

type (
	AaaLogin struct {
		XMLName    struct{} `xml:"aaaLogin"`
		InName     string   `xml:"inName,attr"`
		InPassword string   `xml:"inPassword,attr"`
	}

	AaaLoginResp struct {
		XMLName          struct{} `xml:"aaaLogin"`
		Cookie           string   `xml:"cookie,attr"`
		Response         string   `xml:"response,attr"`
		OutCookie        string   `xml:"outCookie,attr"`
		OutRefreshPeriod string   `xml:"outRefreshPeriod,attr"`
		OutPriv          string   `xml:"outPriv,attr"`
		ErrorCode        int      `xml:"errorCode,attr"`
		ErrorDescr       string   `xml:"errorDescr,attr"`
	}

	ConfigResolveClass struct {
		XMLName        struct{} `xml:"configResolveClass"`
		Cookie         string   `xml:"cookie,attr"`
		InHierarchical string   `xml:"inHierarchical,attr"`
		ClassId        string   `xml:"classId,attr"`
		InFilter       *InFilter
	}

	InFilter struct {
		XMLName xml.Name `xml:"inFilter,omitempty"`
		Eq      *Eq      `xml:"eq,omitempty"`
		Ne      *Ne      `xml:"ne,omitempty"`
		Gt      *Gt      `xml:"gt,omitempty"`
		Ge      *Ge      `xml:"ge,omitempty"`
		Lt      *Lt      `xml:"lt,omitempty"`
		Le      *Le      `xml:"le,omitempty"`
		Wcard   *Wcard   `xml:"wcard,omitempty"`
		Anybit  *Anybit  `xml:"anybit,omitempty"`
		Allbits *Allbits `xml:"allbits,omitempty"`
	}

	// Equality Filter
	Eq struct {
		XMLName  struct{} `xml:"eq"`
		Class    string   `xml:"class,attr"`
		Property string   `xml:"property,attr"`
		Value    string   `xml:"value,attr"`
	}

	// Not Equal Filter
	Ne struct {
		XMLName  struct{} `xml:"ne"`
		Class    string   `xml:"class,attr"`
		Property string   `xml:"property,attr"`
		Value    string   `xml:"value,attr"`
	}

	// Greater Than Filter
	Gt struct {
		XMLName  struct{} `xml:"gt"`
		Class    string   `xml:"class,attr"`
		Property string   `xml:"property,attr"`
		Value    string   `xml:"value,attr"`
	}

	// Greater Than or Equal to Filter
	Ge struct {
		XMLName  struct{} `xml:"ge"`
		Class    string   `xml:"class,attr"`
		Property string   `xml:"property,attr"`
		Value    string   `xml:"value,attr"`
	}

	// Less Than Filter
	Lt struct {
		XMLName  struct{} `xml:"lt"`
		Class    string   `xml:"class,attr"`
		Property string   `xml:"property,attr"`
		Value    string   `xml:"value,attr"`
	}

	// Less Than or Equal to Filter
	Le struct {
		XMLName  struct{} `xml:"le"`
		Class    string   `xml:"class,attr"`
		Property string   `xml:"property,attr"`
		Value    string   `xml:"value,attr"`
	}

	// Wildcard Filter
	Wcard struct {
		XMLName  struct{} `xml:"wcard"`
		Class    string   `xml:"class,attr"`
		Property string   `xml:"property,attr"`
		Value    string   `xml:"value,attr"`
	}

	// Any Bits Filter
	Anybit struct {
		XMLName  struct{} `xml:"anybit"`
		Class    string   `xml:"class,attr"`
		Property string   `xml:"property,attr"`
		Value    string   `xml:"value,attr"`
	}

	// All Bits Filter
	Allbits struct {
		XMLName  struct{} `xml:"allbits"`
		Class    string   `xml:"class,attr"`
		Property string   `xml:"property,attr"`
		Value    string   `xml:"value,attr"`
	}

	ConfigResolveDn struct {
		XMLName        struct{} `xml:"configResolveDn"`
		Cookie         string   `xml:"cookie,attr"`
		InHierarchical string   `xml:"inHierarchical,attr"`
		Dn             string   `xml:"dn,attr"`
	}

	AaaLogout struct {
		XMLName  struct{} `xml:"aaaLogout"`
		InCookie string   `xml:"inCookie,attr"`
	}
)

var (
	ipAddr              string
	queryType           string
	dnOrClass           string
	hierarchical        string
	attributes          string
	expectString        string
	username            string
	password            string
	class               string
	dn                  string
	debug               int
	showEnv             bool
	showVersion         bool
	zeroInst            bool
	proxyString         string
	faultsOnly          bool
	maxTlsVersionString string
	propertyFilter      string
)

func debugPrintf(level int, format string, a ...interface{}) {
	if level <= debug {
		log.Printf(format, a...)
	}
}

func logout(client *http.Client, url, cookie string) {
	xmlAaaLogout := &AaaLogout{InCookie: cookie}
	buf, _ := xml.Marshal(xmlAaaLogout)
	debugPrintf(3, "logout request: %s\n", string(buf))

	data := bytes.NewBuffer(buf)
	resp, err := client.Post(url, "text/xml", data)

	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	debugPrintf(2, "logout respons: %s\n", body)
}

func getXmlAttr(xml_data string, element_name string, attributes []string) (result []string, counter int) {

	counter = 0
	values := make([]string, maxNumAttrib)

	resultStr := ""
	decoder := xml.NewDecoder(bytes.NewBufferString(xml_data))

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			elmt := xml.StartElement(t)
			name := elmt.Name.Local

			if name == element_name {
				counter++
				for _, attr := range token.(xml.StartElement).Attr {
					attr_name := attr.Name.Local
					attr_value := attr.Value
					if i := findIndex(attr_name, attributes); i > -1 {
						values[i] = attr_value
					}
					resultStr = strings.Join(values, ",")
				}
				result = append(result, strings.TrimRight(resultStr, ","))
				resultStr = ""
			}

		}
	}

	return result, counter
}

func findIndex(a string, list []string) int {
	for i, b := range list {
		if b == a {
			return i
		}
	}
	return -1
}

func init() {
	flag.StringVar(&ipAddr, "H", "", "UCS Manager IP address or CIMC IP address")
	flag.StringVar(&queryType, "t", "class", "query type 'class' or 'dn'")
	flag.StringVar(&dnOrClass, "q", "storageLocalDisk", "XML API object class name, examples: storageVirtualDrive or storageLocalDisk or storageControllerProps\nor Distinguished Name (DN) name, examples: \"sys/rack-unit-1\"")
	flag.StringVar(&class, "o", "", "XML API object class name, examples: storageVirtualDrive or storageLocalDisk")
	flag.StringVar(&hierarchical, "s", "false", "true or false. If true, the inHierarchical argument returns all child objects")
	flag.StringVar(&attributes, "a", "id name", "space separated list of XML attributes for display in nagios output and match against *expect* string")
	flag.StringVar(&expectString, "e", "Optimal", "expect string, ok if this is found, examples: 'Optimal' or 'Good' or 'Optimal|Good'")
	flag.StringVar(&username, "u", "", "XML API username")
	flag.StringVar(&password, "p", "", "XML API password")
	flag.IntVar(&debug, "d", 0, "print debug, level: 1 errors only, 2 warnings and 3 informational messages")
	flag.BoolVar(&showEnv, "E", false, "print environment variables for debug purpose")
	flag.BoolVar(&showVersion, "V", false, "print plugin version")
	flag.StringVar(&proxyString, "P", "", "proxy URL")
	flag.BoolVar(&zeroInst, "z", false, "true or false. if set to true the check will return OK status if zero instances where found. Default is false.")
	flag.BoolVar(&faultsOnly, "F", false, "display only faults in output")
	flag.StringVar(&maxTlsVersionString, "M", "1.1", "used TLS version, default: v1.1")
	flag.StringVar(&propertyFilter, "f", "", "property filter <type>:<property>:<value>, works only with query type class (-t class), example: wcard:dn:^sys/chassis-[1-3].*")
}

func main() {
	flag.Parse()

	// send errors to Stdout instead to Stderr
	// http://nagiosplug.sourceforge.net/developer-guidelines.html#PLUGOUTPUT
	log.SetOutput(os.Stdout)
	if showEnv {
		log.Printf("** environment variables start **\n")
		for _, v := range os.Environ() {
			log.Printf("%s\n", v)
		}
		log.Printf("** environment variables end **\n")
	}
	if showVersion {
		fmt.Printf("%s version: %s\n", path.Base(os.Args[0]), version)
		os.Exit(0)
	}
	attributeArray := strings.Split(attributes, " ")
	attributeDescr := strings.Replace(attributes, " ", ",", -1)

	debugPrintf(3, "attributes: %v\n", attributeArray)

	if len(attributeArray) > maxNumAttrib {
		fmt.Printf("maximum number of attibutes is %d\n", maxNumAttrib)
		os.Exit(3)
	}

	output := "Cisco UCS "
	output += dnOrClass
	output += " (" + attributeDescr + ")"

	switch queryType {
	case "class":
		class = dnOrClass
		debugPrintf(2, "query type: class (%s)\n", class)
	case "dn":
		dn = dnOrClass
		debugPrintf(2, "query type: dn (%s)\n", dn)
	}

	debugPrintf(1, "ip addr: %s dn or class: %s\n", ipAddr, dnOrClass)
	debugPrintf(1, "hierarchical: %s attributes: \"%s\" expectString: %s\n", hierarchical, attributes, expectString)

	var maxTlsVersion uint16

	maxTlsVersion = tls.VersionTLS11
	if maxTlsVersionString == "1.2" {
		maxTlsVersion = tls.VersionTLS12
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MaxVersion:         maxTlsVersion,
			},
		},
	}

	url := "https://" + ipAddr + "/nuova"
	debugPrintf(2, "url: %s\n", url)
	xml_aaaLogin := &AaaLogin{InName: username, InPassword: password}
	buf, _ := xml.Marshal(xml_aaaLogin)
	debugPrintf(3, "login request: %s\n", string(buf))
	data := bytes.NewBuffer(buf)
	resp, err := client.Post(url, "text/xml", data)

	if err != nil {
		debugPrintf(3, "login error: %s\n", err.Error())
		if strings.Contains(err.Error(), "EOF") {
			fmt.Printf("CRIT: EOF received from the target system.\n")
		} else {
			fmt.Printf("CRIT: %v\n", err)
		}
		os.Exit(3)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	debugPrintf(2, "http status code: %s\n", resp.Status)
	debugPrintf(3, "login response: %s\n", string(body))

	xmlAaaLoginResp := &AaaLoginResp{Cookie: "", Response: "", OutCookie: "", OutRefreshPeriod: "", OutPriv: ""}

	err = xml.Unmarshal([]byte(body), &xmlAaaLoginResp)

	if err != nil {
		if strings.Contains(err.Error(), "EOF") {
			fmt.Printf("CRIT: EOF received from the target system. Check if CIMC interface is working.\n")
		} else {
			fmt.Printf("CRIT: %v\n", err)
		}
		os.Exit(3)
	}

	defer logout(client, url, xmlAaaLoginResp.OutCookie)

	debugPrintf(2, "%#v\n", xmlAaaLoginResp)

	debugPrintf(1, "login cookie: %s\n", xmlAaaLoginResp.OutCookie)
	debugPrintf(3, "login error code: %d\n", xmlAaaLoginResp.ErrorCode)

	if xmlAaaLoginResp.ErrorCode != 0 {
		fmt.Printf("aaaLogin Error: %s (%d)\n", xmlAaaLoginResp.ErrorDescr, xmlAaaLoginResp.ErrorCode)
		os.Exit(3)
	}

	num_found := 0

	switch queryType {
	case "class":
		xmlConfigResolveClass := &ConfigResolveClass{Cookie: xmlAaaLoginResp.OutCookie, InHierarchical: hierarchical, ClassId: class}
		if len(propertyFilter) > 0 {
			parts := strings.Split(propertyFilter, ":")
			debugPrintf(3, "propertyFilter split: %#v\n", parts)
			switch parts[0] {
			case "eq":
				xmlConfigResolveClass.InFilter.Eq = &Eq{Class: class, Property: parts[1], Value: parts[2]}
			case "ne":
				xmlConfigResolveClass.InFilter.Ne = &Ne{Class: class, Property: parts[1], Value: parts[2]}
			case "gt":
				xmlConfigResolveClass.InFilter.Gt = &Gt{Class: class, Property: parts[1], Value: parts[2]}
			case "ge":
				xmlConfigResolveClass.InFilter.Ge = &Ge{Class: class, Property: parts[1], Value: parts[2]}
			case "lt":
				xmlConfigResolveClass.InFilter.Lt = &Lt{Class: class, Property: parts[1], Value: parts[2]}
			case "le":
				xmlConfigResolveClass.InFilter.Le = &Le{Class: class, Property: parts[1], Value: parts[2]}
			case "wcard":
				xmlConfigResolveClass.InFilter.Wcard = &Wcard{Class: class, Property: parts[1], Value: parts[2]}
			case "anybit":
				xmlConfigResolveClass.InFilter.Anybit = &Anybit{Class: class, Property: parts[1], Value: parts[2]}
			case "allbits":
				xmlConfigResolveClass.InFilter.Allbits = &Allbits{Class: class, Property: parts[1], Value: parts[2]}
			}
		}

		debugPrintf(3, "xmlConfigResolveClass request: %#v\n", xmlConfigResolveClass)

		buf, err = xml.MarshalIndent(xmlConfigResolveClass, "  ", "    ")
		if err != nil {
			debugPrintf(2, "xmlConfigResolveClass marshal error: %s\n", err)
		}

		debugPrintf(3, "buf before regex:\n%s\n", string(buf))

		// see issue:
		// encoding/xml: cannot marshal self-closing tag #21399
		// https://github.com/golang/go/issues/21399
		re := regexp.MustCompile("></.*?>")
		result := re.ReplaceAllString(string(buf), " />")
		data = bytes.NewBuffer([]byte(result))
		debugPrintf(3, "configResolveClass request:\n%s\n", result)
		resp, err = client.Post(url, "text/xml", data)
		if err != nil {
			fmt.Printf("error: %v", err)
			os.Exit(3)
		}
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		debugPrintf(2, "configResolveClass respons: %s\n", body)

	case "dn":
		xmlConfigResolveDn := &ConfigResolveDn{Cookie: xmlAaaLoginResp.OutCookie, InHierarchical: hierarchical, Dn: dn}

		buf, err = xml.Marshal(xmlConfigResolveDn)
		if err != nil {
			log.Printf("xmlConfigResolveDn marshal error: %s\n", err)
		}
		debugPrintf(3, "configResolveDn request: %s\n", string(buf))
		data = bytes.NewBuffer(buf)
		resp, err = client.Post(url, "text/xml", data)
		if err != nil {
			fmt.Printf("error: %v", err)
			os.Exit(3)
		}
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		debugPrintf(2, "configResolveDn respons: %s\n", body)

	}

	// "defer logout" not working ? ... so:
	logout(client, url, xmlAaaLoginResp.OutCookie)

	r, n := getXmlAttr(string(body), class, attributeArray)
	debugPrintf(3, "result: %v counter: %d\n", r, n)

	re := regexp.MustCompile(expectString)

	debugPrintf(3, "\n%v\n\n", r)
	for _, val := range r {
		n := len(re.FindAllString(val, -1))
		num_found += n
		debugPrintf(3, "%s num_found=%d n=%d", val, num_found, n)
		if n == 0 && faultsOnly {
			output += "\n" + val
		}
		if !faultsOnly {
			output += "\n" + val
		}

	}

	prefix := "UNKNOWN"
	ret_val := 3

	// new in version 0.9: output example for case (zeroInst && num_found == 0 && n == 0) ---> "... (0 of 0 ok)" or "... (<num_found> of <n> ok)"
	if (zeroInst && num_found == 0 && n == 0) || (n > 0 && num_found == n) {
		prefix = "OK"
		ret_val = 0
	} else {
		prefix = "CRIT"
		ret_val = 2
	}

	fmt.Printf("%s - %s (%d of %d ok)\n", prefix, output, num_found, n)
	os.Exit(ret_val)
}
