package crypto

import (
	"crypto/aes"
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/alecthomas/unsafeslice"
	"github.com/twmb/murmur3"
)

type pseudorandomCodeTest struct {
	out         string // hex
	in          string
	hashedBlock string // hex
	aesKey      string // hex
	hIdx        byte
}

var aesTestStrings = []pseudorandomCodeTest{
	{"7d040a001a48af28aa3a1837ed04864935f4b73a9a54ad7c0decd361f2f6ee30590c4a43c61873cc39c45be00ac71d31b0cab6d39e167971622aa3ed41c8b406", "", "00000000000000000000000000000000", "6680dc641356cbdb590c370f747d4e9f", 0},
	{"7c9c0421edbb2cb9583fe0af30a82b21c9424a31af3d13c092edb0fefbd9e2a23cb6969021c2e42f927c52d479957a18cf98b6c041d2a3620e4a690ad401cc2b", "", "b55cff6ee5ab10468335f878aa2d6251", "6680dc641356cbdb590c370f747d4e9f", 1},
	{"43757caabedfe32a7d1aad2a1966633e11fde0067d2666a96d2a65fb8e8cb62e167c2c3372778ac3c3ebea4c5c495b5bd30f3bc53e3d26c314cc04d8d18fdf26", "", "e85d6c701ab844305ba3e837cc6bf918", "6680dc641356cbdb590c370f747d4e9f", 2},
	{"04aaa630968b3e58bd381208856009de37a1a629b99f68007a1b060f8439d2e19b4c858dd1be11cced3c9a82bfb135b5f24d96da6c0aeb141d862f164b631c25", "", "483771bfd14dd5791d2966b7f1e7f50a", "6680dc641356cbdb590c370f747d4e9f", 3},
	{"b6d67803e1154d2afcca1906b9e72f848c454ab1345d098d78d798aae8f6eedf85b2f20fc28e9f590e3ad5dce9a49f39ed5dba3f1166023c3fc25bb090527ec7", "", "00000000000000000000000000000000", "5f2d0b92a398c22fb9816d0de476db2c", 0},
	{"38201a5379641fcdede51196fa2a3be8aef7ce5dbb33db95c77751414085128d260fb4c8acc6cd5f522f932dded2a025d60984445a12123274f8b137be2a74a5", "", "b55cff6ee5ab10468335f878aa2d6251", "5f2d0b92a398c22fb9816d0de476db2c", 1},
	{"f4c9a342bc66ee04dc517d604564faa96942f5ada68b2ff8e83cd33eb362761a199f3e368a854d3a6dc7be9385a5035bc8f36c1f99276981ef6fdc20dc2bc8dd", "", "e85d6c701ab844305ba3e837cc6bf918", "5f2d0b92a398c22fb9816d0de476db2c", 2},
	{"aa843dd67a9ef7bb501f53b0ab18f013e39da3aee28b0646016420e13a28e0e476b31355e862544b6c3ec8350f69d321628a125c6481efbb2aa91d492708b180", "", "483771bfd14dd5791d2966b7f1e7f50a", "5f2d0b92a398c22fb9816d0de476db2c", 3},
	{"98c919064c1023e35bcf4b5e326dc339dd5373848e06c37063fb5c737c9207a3fcc5f740909170a94eee137a1850b11ae32adc1c1b51074bd9c71af701fb390f", "", "00000000000000000000000000000000", "6f612185e1b2d64a0657fc056e156a89", 0},
	{"b3ac585807ec5666c63071e712c169f43f207cc15cd40cf7ab0d7dcb992c81074d6a85079291daf34ddbbbe355fe634d4c3a9cf8a81fca9e637fbc30dc0299dd", "", "b55cff6ee5ab10468335f878aa2d6251", "6f612185e1b2d64a0657fc056e156a89", 1},
	{"9ae39eb0b7ad0053f44cf6840ef7b2fa139cdc50a3efeeee2958c11c78f83527d255b01fe0d12100a299bb41b568922a96e03fa842b8c1b0bb93da8149d1b138", "", "e85d6c701ab844305ba3e837cc6bf918", "6f612185e1b2d64a0657fc056e156a89", 2},
	{"07f4746c6ebc748052ca6f6356794c4738df720874600ebab6bb09772f8e4c2e916bbe3aeebce61b38d0c15a57a18daf49a25dd70a504548bff7ca135ed88f76", "", "483771bfd14dd5791d2966b7f1e7f50a", "6f612185e1b2d64a0657fc056e156a89", 3},
	{"473a27dc50637a128be257d46864333862c12ab5b4280a50f0523ebfd91cc00be773f3faba976a0a6dab807a4b62b848ebd680b42ff8fed6eb5bf2f15f4ce144", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "9b1f53bcdf72ab1dc9401ce0bc4fdf89", "6680dc641356cbdb590c370f747d4e9f", 0},
	{"3271e4ba95d28b07c19849cad13e1ec6a686038e98335b2323cc0c7d55beb031474ad0c0a2d93eb5765c319f2af5c70e948a640e8e2f612ed43beb07ffe4abc4", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "1a674788b078d45d4e77a5448299b159", "6680dc641356cbdb590c370f747d4e9f", 1},
	{"ec6917ea8f475d915fbf944fce360da40597f7c5c6489e587c8dfd53c5528c775e26839a952a29805de742b36d6e0141c7d5a3bc9e6b1f927f6318932ad148f2", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "2bc4ee043058967d689b039dc71cf451", "6680dc641356cbdb590c370f747d4e9f", 2},
	{"20d77c710d53437a56ad0ec8541271916ea92d8458b8013bc6a2579afae606af0026fc85363e177aa243ac9cf6d1a64f4da7b24cb9243e8e5c7949af1b6a9d2e", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "fef29399b3f96089429afa19e8048f93", "6680dc641356cbdb590c370f747d4e9f", 3},
	{"f8d545b70906d5ffcedec23901b8f664e2a7c4936704ec79aaad8690b83ed06c475576a0398dc34d3495e9a058f88b655c3e770e01e6154657e76f57cc6d985c", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "9b1f53bcdf72ab1dc9401ce0bc4fdf89", "5f2d0b92a398c22fb9816d0de476db2c", 0},
	{"7a82c33c1f4d4e47bd232e781e97a08bf4a0742a56419d6702a4dbd31eaaaf479afb1ac8bb52342ebdd76e95b1b56dc640a8c46dd138a3137db4928b6569c90c", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "1a674788b078d45d4e77a5448299b159", "5f2d0b92a398c22fb9816d0de476db2c", 1},
	{"634f08d0ec1096f745e858f004d8aed9da7c239676a5662a6abdc63a55c5b78b85fa9879d939a37599be5c0cd85e4e630b981889b6fba80142f42a1179cd0351", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "2bc4ee043058967d689b039dc71cf451", "5f2d0b92a398c22fb9816d0de476db2c", 2},
	{"a8fd8a5393a0ae669ed82eea4b1640d794166266d1aad5c3f50e9ee09373569e077e7b779d7d7f2799b643364cfa51f3105a92bed83faa8efab26cfd6a6f9e0d", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "fef29399b3f96089429afa19e8048f93", "5f2d0b92a398c22fb9816d0de476db2c", 3},
	{"912ab94cb0dba6ef52548502915de1ab467bf72ba74e9594f97d3bc0fc35ac1c716e8a2e2047b7f2ce2029d3a54355eeaf809360e8acf9461bf99cd7e6450d1b", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "9b1f53bcdf72ab1dc9401ce0bc4fdf89", "6f612185e1b2d64a0657fc056e156a89", 0},
	{"8b76c6654dfb5acfbe99ed7e9fa0914579b44b4409cfe15e3d5b9b1cc9943e3a1eb34a1b4b3f8eb01f5b09d667ab44e8f05cff88c18ddc42a14d25a1ad0be605", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "1a674788b078d45d4e77a5448299b159", "6f612185e1b2d64a0657fc056e156a89", 1},
	{"c63b00c909dfcf2490066de4de4341c7fdeafda48513121b75ac6c32ed8ed4ebc239ebe444884431f95baa3367c2d68be35d6402dc4337f840306d6c731c05ac", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "2bc4ee043058967d689b039dc71cf451", "6f612185e1b2d64a0657fc056e156a89", 2},
	{"c06f380eeb2cd7e93f16f215f739650bd98253b3234d4a1b6b5878eb2c8b8bd3b117d4a78a4faad53a6fc35cb5d5701669884950bbda57338b7b027ebba32bc8", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "fef29399b3f96089429afa19e8048f93", "6f612185e1b2d64a0657fc056e156a89", 3},
	{"2b13d5c18268b7f7b432a36e8c4d255a1371ca0ccac3a712a6f8d048dbcb2306179d458987a595287be7cb97ece82cb6e3ae7420cfc8ea90572b29c8ab750634", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "2668543c024fc0d60c602ab367634065", "6680dc641356cbdb590c370f747d4e9f", 0},
	{"c8a12ca9be5bd3215ea53e78c0be09b409f161723544b29c94960782fdff206557ae579d44a380bb34f3bc3edeaf1a30050781e4dae459176ddc56f2bca7a27d", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "d664b0bc17a117fabf3350b251b11b11", "6680dc641356cbdb590c370f747d4e9f", 1},
	{"d1370dbe6544cf2fc35d8f0fbe48c86516df3d3f97bb87f0227778823eabba19cf4be97208f88d58d9561876e9f706cb108552c7e5c049ff318ce8dc2fe9c163", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "bb5d1d8157b0f209c98d407ca7d20424", "6680dc641356cbdb590c370f747d4e9f", 2},
	{"7b1dac8f699e6d1e1c8a7d490b26bc12e47f219d2cf257504ade2657286ad2e8db4fd4b9cf5470e77a3622d64718c058809fa551fc2791734419dedebc0f4c7f", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "2daf7ccbc24cbc8426034d78b03e6208", "6680dc641356cbdb590c370f747d4e9f", 3},
	{"bb381f67c8ddc711d2d45d6bf46d09390e6888c36283773cad7c954585dfd2ddc572dc59397099eb6eab3d9b68a9be7f17f309e9e655b52762136a01fd871acf", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "2668543c024fc0d60c602ab367634065", "5f2d0b92a398c22fb9816d0de476db2c", 0},
	{"3f6ac072b23dac2eed840a855734527fb4b293a3bcd79a9344e39810d39308d964615050ce9ca605fa2ed50b38812da96485cd6ad54d0f531c70dd95208de6f6", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "d664b0bc17a117fabf3350b251b11b11", "5f2d0b92a398c22fb9816d0de476db2c", 1},
	{"98cf45e94daa57c637e32436ab81fe8cfab8f96de55a085840fc546c5299fe0e0b4b7027a62de87a90200e790f0c9b2e2cae418a0a934dacd9ccbc634431bf25", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "bb5d1d8157b0f209c98d407ca7d20424", "5f2d0b92a398c22fb9816d0de476db2c", 2},
	{"8fb793ca77ba1b033876ce7d555ec7188278dc11b6d2e53471ff5d7171cf52ed7c36a581e2204e227f08abdc0d534a3885c585bbd289d022404900ef68efd771", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "2daf7ccbc24cbc8426034d78b03e6208", "5f2d0b92a398c22fb9816d0de476db2c", 3},
	{"762fef8c12ec980a70f0b89d11490a0016d9b385a7414d9dc24b21cc9ab0285a79cc7ef346f6562beca5c597ed40363d390c865b9cbfd78ad1096a13fa4c67f0", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "2668543c024fc0d60c602ab367634065", "6f612185e1b2d64a0657fc056e156a89", 0},
	{"a9046d7afd68be3dc45447c42e644d16114d3b09c111184a236c58d58da7b04011f9a522e6ca8df776130faf0170a8ad7a79e064779508ed9440b30e0b68d1a3", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "d664b0bc17a117fabf3350b251b11b11", "6f612185e1b2d64a0657fc056e156a89", 1},
	{"ca17a038d904817d3fd01131557d301048ac80cd1a42627281fd159c657fa5ce737736be000a823ffc1443caf6de93d6f084314386270a851e76da38a8dd1b3d", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "bb5d1d8157b0f209c98d407ca7d20424", "6f612185e1b2d64a0657fc056e156a89", 2},
	{"7a7f4a139f07d13aa12130c0d8d992f090f21e7c60a41dd632cd8e0a79e35402fef66d83168805629e737943c352ebda30a724d68e9c483f6827a0cf6cc4c087", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "2daf7ccbc24cbc8426034d78b03e6208", "6f612185e1b2d64a0657fc056e156a89", 3},
	{"400a4ad961e475d8c2e20f07784a02634a9d10f2f7d81e6add27c60f62e03b06027459b547ac5ebf68bd01838d38c9ec93a5d44a9fe0945eb5d298fe882e6582", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "c8e187160fcf2b834c035bd118c3311d", "6680dc641356cbdb590c370f747d4e9f", 0},
	{"02818be2a3c5d4fe10bfae41112ea228755d3e93d5aeb951bb6f79b754f01c176caeae76afff7cfd13b8d10b095e40ddde2bc342c69cd05ce52fb686dd1d761e", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "33e0f08d32a2be474196c1ddf6383357", "6680dc641356cbdb590c370f747d4e9f", 1},
	{"44361dde703bcf5291bc538a396dcacbcebad3562bb16e58711573fea60b0eae67c9bfe64ce41f157b5442cc64f06e299df522ae86966eda86845ba057b22c30", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "039c1ebac2c56ceb0b3c1febc88f191e", "6680dc641356cbdb590c370f747d4e9f", 2},
	{"1368587df1fbd0ae5b02586f08bfbc9a8526bc5ea59f6f5895e05d6e19bed085ae8dd10aa829d04b2067d0ec1d1a99a1dc73492dbe64bad76dc9796557dac831", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "e5504244ce01b922f5fd6c23e432b5b5", "6680dc641356cbdb590c370f747d4e9f", 3},
	{"3abc9c2f7c33fce44ddb3f38fba586da04f9dbc77f1319caebdfa7bd9b4fb7cd5e20c3e84f9fc91dd1ca164ec466ba47208ef01174ff23cbd9223a3e8f39a5ea", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "c8e187160fcf2b834c035bd118c3311d", "5f2d0b92a398c22fb9816d0de476db2c", 0},
	{"e37f360d2a8476a9fc485cff64130f9dcb0e6aae4be970fd21c5e0022869826a31fc744523e93690470beabce06ff705019aae108b6deb6c1998ec6f2dade395", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "33e0f08d32a2be474196c1ddf6383357", "5f2d0b92a398c22fb9816d0de476db2c", 1},
	{"c8c31f8c69c66db28ef1108261c372ef66cd5bb1ae099acf017bbda14a4dbbc8530a16b6edde0d0c52fa9087e1511597eeb59b97c9209baeb6d3aef14f9f37bb", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "039c1ebac2c56ceb0b3c1febc88f191e", "5f2d0b92a398c22fb9816d0de476db2c", 2},
	{"edd3538741f35d76f97cee03f3f14b289a4264f52192300b5f2c8744c80b61b9bda7c2e0da48ce0692c2fcd22d6361bf7b96a19b8e311995d3fab9e9cab37553", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "e5504244ce01b922f5fd6c23e432b5b5", "5f2d0b92a398c22fb9816d0de476db2c", 3},
	{"ac80c57cebd18facf12b979349029964c3cf512d080bed818cb9ec42a4c64d782af8b36e4a39ecd65fbb3a6f4e12c8799dfd928af589e83fad1e313020c76d7a", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "c8e187160fcf2b834c035bd118c3311d", "6f612185e1b2d64a0657fc056e156a89", 0},
	{"43704b603a76c16e08e69546dbff0b57fdc8d5d4cb67588b4a52675f7fb327e1d5ee565dd0b58100f4d14f9deb7fafe9ff13fda30b80a4f196021b20d17b50fe", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "33e0f08d32a2be474196c1ddf6383357", "6f612185e1b2d64a0657fc056e156a89", 1},
	{"b8831aa6cccefe526c1054701da0268b4f6ea8e2ae5418b2df976ca4e924c0ae75d7dc901dc64bc53cb69a8483b5c51a47ab0043f018f23e575ff470669697e3", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "039c1ebac2c56ceb0b3c1febc88f191e", "6f612185e1b2d64a0657fc056e156a89", 2},
	{"86156e15c1e6511d2627384d76d38f3f3e3402be0f34c901afb10ea118d5fb6b400a07c5b93e06f5a15a710731e1a694b1e21248ee830bc0774a400312881e7c", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "e5504244ce01b922f5fd6c23e432b5b5", "6f612185e1b2d64a0657fc056e156a89", 3},
}

type xorCipherTest struct {
	cipherText string // hex
	plainText  string
	xorKey     string // hex
	choice     byte
}

var xorCipherTestStrings = []xorCipherTest{
	{"32c207fb7951ac2f8edb334b120f6337279f19af323b27be976e520796a8f7499e420d67ec78c58ce0d34274cc7ef46b2d5b16d0a5d012452b", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "3d1fe501030418491fde1223b3cf05094996fe655139934b538095715b7c68d5", 0},
	{"e2c4bf84b04050e41983d021c91254cd38a0cbf26abaf2a863a102a8b33b9f48c5a033df4a68b1bcaa46acd3d6abebbe113d18c664f22feabd", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "3d1fe501030418491fde1223b3cf05094996fe655139934b538095715b7c68d5", 1},
	{"ea70ec5f79ce3df7c390e804f5940198fb33f64094a13354e9fd8881f85cb85047674d24010576753f7457f0ee4e0da927a129e2406c986edc", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "4a8ec3b4f381da6b8ad2ad122ea71ea4064b90e280f65075e676feb9c40806c1", 0},
	{"388b761ddb6c9ea235cc63862fa1f4cb4aa1e71d2d80010720067184fea081ca03c0dd231febff6d84bb564d5992f663edeba580fd09970ddb", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "4a8ec3b4f381da6b8ad2ad122ea71ea4064b90e280f65075e676feb9c40806c1", 1},
	{"4ef613000dc9612bead97ef1c802233dc311ecfb33ae2c9e063e9eebc939dc9740ad7adea498debe7000ebc928f2008ff9b7c86605959c0a8b", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "2e87ae384abbb3b5d593385100bcee15a25766097128e4930353b88dc2a5e328", 0},
	{"0d729e57468445fe8aa5b7344d0dc330822fab5efc27237b6295a4ebfa9cd3b9553d6ada3f6a6fa0cb0ec2903589ecb6a47ee04cdde8e86bc7", "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "2e87ae384abbb3b5d593385100bcee15a25766097128e4930353b88dc2a5e328", 1},
	{"118a5bfd6910dc6bde892505374d22747e8c50ee3c2c5de9d67f1902ccbbe745ca431567b73c8997f18f41308d32d57a3b0441c0c281440077f2c2d29e1c5749ac00", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "3d1fe501030418491fde1223b3cf05094996fe655139934b538095715b7c68d5", 0},
	{"c18ce382a00120a049d1c66fec50158e61b382b364ad88ff22b049ade9288f4491a12bdf112cfda7bb1aaf9797e7caaf07624fd603a379afe12cc258e2a7ea9e0d3f", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "3d1fe501030418491fde1223b3cf05094996fe655139934b538095715b7c68d5", 1},
	{"c938b059698f4db393c2fe4ad0d640dba220bf019ab64903a8ecc384a24fa85c136655245a413a6e2e2854b4af022cb831fe7ef2273dce2b808c6df2dd5835923ad1", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "4a8ec3b4f381da6b8ad2ad122ea71ea4064b90e280f65075e676feb9c40806c1", 0},
	{"1bc32a1bcb2deee6659e75c80ae3b58813b2ae5c23977b5061173a81a4b391c657c1c52344afb37695e7550918ded772fbb4f2909a58c148871eea941d34093bb002", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "4a8ec3b4f381da6b8ad2ad122ea71ea4064b90e280f65075e676feb9c40806c1", 1},
	{"6dbe4f061d88116fba8b68bfed40627e9a02a5ba3db956c9472fd5ee932acc9b14ac62deffdc92a5615ce88d69be219eefe89f7662c4ca4fd71916c06d28c0626d05", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "2e87ae384abbb3b5d593385100bcee15a25766097128e4930353b88dc2a5e328", 0},
	{"2e3ac25156c535badaf7a17a684f8273db3ce21ff230592c2384efeea08fc3b5013c72da642e23bbda52c1d474c5cda7b221b75cbab9be2e9b38439239b5446cdfa7", "e:9c1a66577adb510cf5a7763bdc5a05d17e648b16b62ccdd260497394536662d9", "2e87ae384abbb3b5d593385100bcee15a25766097128e4930353b88dc2a5e328", 1},
	{"20d807be3e048d3c88d7661d734071652fcf55b433681eb69168180f8dfabe1e8e13026fe870c580b2dc0369d971d17f2c5304d097cc53526ea19e97cd44410fad5467fc8334b777404a59242f86a8f7ecb27ba243eb0d89537dcae3dfa701ebdeabca4a7c814763fa4b556325f490687e1435d0e380e35e4321cd82ac51bd53fc94f84f", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "3d1fe501030418491fde1223b3cf05094996fe655139934b538095715b7c68d5", 0},
	{"f0debfc1f71571f71f8f8577a85d469f30f087e96be9cba065a748a0a869d61fd5f13cd74e60b1b0f849edcec3a4ceaa10350ac656ee6efdf87f9e1db1fffcd80c6b65a59651e0e7bda457b1317369b5189753b0e42a799d22d57df3d8539614b53ff0eb81f6f7149dd57b79f55b2618b962c6ea30e99469e28d51166698af1820137a8d", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "3d1fe501030418491fde1223b3cf05094996fe655139934b538095715b7c68d5", 1},
	{"f86aec1a3e9b1ce4c59cbd5294db13caf363ba5b95f20a5ceffbc289e30ef1075736422c050d76796d7b16edfb4128bd26a93be27270d97999df31b78e0023d43b857280aab29764e65e2b83bbf7a4ebcad4c9c40050ba8200bfe613588506dd2063d90bae16637d4ea237d77ae85b661f3fa7650bfecb1a6ba1d705ebe347ec3f9d6a08", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "4a8ec3b4f381da6b8ad2ad122ea71ea4064b90e280f65075e676feb9c40806c1", 0},
	{"2a9176589c39bfb133c036d04eeee69942f1ab062cd3380f26003b8ce5f2c89d1391d22b1be3ff61d6b417504c9dd377ece3b780cf15d61a9e4db6d14e6c1f7db156fbab84a5cc4dd648ba58582611d813a38c96f627b35a0404178c6653a198fbf696193fc2c88e582faf1370c16885a13d09ecc3a28770b08cb96f97b19aa0215ff0c7", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "4a8ec3b4f381da6b8ad2ad122ea71ea4064b90e280f65075e676feb9c40806c1", 1},
	{"5cec13454a9c4038ecd52ba7a94d316fcb41a0e032fd15960038d4e3d26b95c050fc75d6a090deb2220faad43dfd259bf8bfda663789dd1dce4a4a853e70d6246c51614bf4d45c82dbb5b5b12385a629efc4d14a249cb280c18b4e57e35c1366fc1b8db9ed1b1bbafc8c681195b82a25c074a729995693e17ff973fdfa0ebaedd4f624d6", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "2e87ae384abbb3b5d593385100bcee15a25766097128e4930353b88dc2a5e328", 0},
	{"1f689e1201d164ed8ca9e2622c42d1628a7fe745fd741a736493eee3e1ce9aee456c65d23b626fac9901838d2086c9a2a576f24ceff4a97c826b1fd76aed522adef31a71971fe9349567d1865f31a146c2829818c20a10b572782b59830ddd40b37817c22988d95965a50a52cecccf89a4ebe8e34922ea1feeb643786dee8fc579a663fe", "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "2e87ae384abbb3b5d593385100bcee15a25766097128e4930353b88dc2a5e328", 1},
}

func TestMurmur3Hashing(t *testing.T) {
	for _, s := range aesTestStrings {
		lo, hi := murmur3.SeedSum128(uint64(s.hIdx), uint64(s.hIdx), []byte(s.in))
		h := unsafeslice.ByteSliceFromUint64Slice([]uint64{lo, hi})
		if s.hashedBlock != fmt.Sprintf("%x", h) {
			t.Errorf("murmur3 hashing did not return expected result with hash index %v for\ninput: %v\nreturned: %x\nexpected: %v", s.hIdx, s.in, string(h), s.hashedBlock)
		}
	}
}

func TestPseudorandomCode(t *testing.T) {
	for _, s := range aesTestStrings {
		aesKey, err := hex.DecodeString(s.aesKey)
		if err != nil {
			t.Fatal(err)
		}

		aesBlock, err := aes.NewCipher(aesKey)
		if err != nil {
			t.Fatal(err)
		}
		enc := PseudorandomCode(aesBlock, []byte(s.in), s.hIdx)
		if s.out != fmt.Sprintf("%x", enc) {
			t.Errorf("AES block encoding did not return expected result with hash index %v for\ninput: %v\nreturned: %x\nexpected: %v", s.hIdx, s.in, string(enc), s.out)
		}
	}
}

func TestEncryptionWithXorCipherWithBlake3(t *testing.T) {
	for _, s := range xorCipherTestStrings {
		xorKey, err := hex.DecodeString(s.xorKey)
		if err != nil {
			t.Fatal(err)
		}

		cipherText, err := XorCipherWithBlake3(xorKey, s.choice, []byte(s.plainText))
		if err != nil {
			t.Fatal(err)
		}
		if s.cipherText != fmt.Sprintf("%x", cipherText) {
			t.Fatalf("Encryption via XOR cipher with Blake 3 did not return expected result with choice bit %v for\ninput: %v\nreturned: %x\nexpected: %v", s.choice, s.plainText, string(cipherText), s.cipherText)
		}
	}
}

func TestDecryptionWithXorCipherWithBlake3(t *testing.T) {
	for _, s := range xorCipherTestStrings {
		xorKey, err := hex.DecodeString(s.xorKey)
		if err != nil {
			t.Fatal(err)
		}

		cipherText, err := hex.DecodeString(s.cipherText)
		if err != nil {
			t.Fatal(err)
		}
		plainText, err := XorCipherWithBlake3(xorKey, s.choice, cipherText)
		if err != nil {
			t.Fatal(err)
		}
		if s.plainText != string(plainText) {
			t.Fatalf("Decryption via XOR cipher with Blake 3 did not return expected result with choice bit %v for\ninput: %v\nreturned: %x\nexpected: %v", s.choice, s.cipherText, string(plainText), s.plainText)
		}
	}
}

func BenchmarkPseudorandomCode(b *testing.B) {
	// the normal input is a 64 byte digest with a byte indicating
	// which hash function is used to compute the cuckoo hash
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	in := make([]byte, 64)
	aesKey := make([]byte, 16)
	prng.Read(in)
	prng.Read(aesKey)

	aesBlock, err := aes.NewCipher(aesKey)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PseudorandomCode(aesBlock, in, 0)
	}
}

func BenchmarkEncryptionWithXorCipherWithBlake3(b *testing.B) {
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	xorKey := make([]byte, 32)
	p := make([]byte, 64)
	prng.Read(xorKey)
	prng.Read(p)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := XorCipherWithBlake3(xorKey, 0, p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecryptionWithXorCipherWithBlake3(b *testing.B) {
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	xorKey := make([]byte, 32)
	p := make([]byte, 64)
	prng.Read(xorKey)
	prng.Read(p)

	c, err := XorCipherWithBlake3(xorKey, 0, p)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := XorCipherWithBlake3(xorKey, 0, c); err != nil {
			b.Fatal(err)
		}
	}
}
