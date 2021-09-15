const Avalanche = require('avalanche'); 
const bintools = new Avalanche.BinTools()
let toEncode = process.argv[2]
if (toEncode.length > 32) {
  console.log('Name must be 32 characters or less')
} else {
  console.log(`encoding ${toEncode}`)
  while (toEncode.length < 32) {
    toEncode += '\u0000'
  }
  const input = Buffer.from(toEncode, 'utf8')
  console.log(bintools.cb58Encode(input))
}
