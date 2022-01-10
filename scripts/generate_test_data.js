const fs = require('fs');
const faker = require('faker');
const XLSX = require('xlsx');
const CSV = require('papaparse');
const parquet = require('@dsnp/parquetjs');

const data = [];
for (let i = 0; i < 1000; i++) {
  data.push({
    ' Name ': faker.name.findName(),
    'Phone Number ': faker.phone.phoneNumber(),
    Email: faker.internet.email(),
    Street: faker.address.streetAddress(),
    '    City ': faker.address.city(),
    State: faker.address.state(),
    'Zip Code ': faker.address.zipCode(),
    'Routing Number   ': faker.finance.routingNumber(),
    Department: faker.commerce.department(),
    'Company\t': faker.company.companyName(),
    'Created At ': faker.date.past(),
    'Profile Photo': faker.image.imageUrl(),
    '  Description': faker.lorem.paragraph(),
    Activated: faker.datatype.boolean(),
  });
}
console.log(`Generated ${data.length} test data rows`);

const directory = 'testdata/';

async function write() {
  // Write as CSV
  const csvname = directory + 'userdata.csv';
  fs.writeFileSync(csvname, CSV.unparse(data));
  console.log(`Wrote ${csvname}`);

  // Write as TSV
  const tsvname = directory + 'userdata.tsv';
  fs.writeFileSync(
    tsvname,
    CSV.unparse(data, {
      delimiter: '\t',
    })
  );
  console.log(`Wrote ${tsvname}`);

  // Write as JSON
  const jsonname = directory + 'userdata.json';
  fs.writeFileSync(jsonname, JSON.stringify(data));
  console.log(`Wrote ${jsonname}`);

  // Write as JSON lines
  const jsonlinesnames = directory + 'userdata.jsonl';
  fs.writeFileSync(
    jsonlinesnames,
    data.map((row) => JSON.stringify(row).replace(/\n/g, '')).join('\n')
  );
  console.log(`Wrote ${jsonlinesnames}`);

  // Write as .ods file
  const ws = XLSX.utils.json_to_sheet(data);
  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, 'Sheet 1');
  const odsname = directory + 'userdata.ods';
  XLSX.writeFile(wb, odsname);
  console.log(`Wrote ${odsname}`);

  // Write as Excel file
  const excelname = directory + 'userdata.xlsx';
  XLSX.writeFile(wb, excelname);
  console.log(`Wrote ${excelname}`);

  // Write as parquet file
  const schema = new parquet.ParquetSchema({
    ' Name ': { type: 'UTF8' },
    'Phone Number ': { type: 'UTF8' },
    Email: { type: 'UTF8' },
    Street: { type: 'UTF8' },
    '    City ': { type: 'UTF8' },
    State: { type: 'UTF8' },
    'Zip Code ': { type: 'UTF8' },
    'Routing Number   ': { type: 'INT64' },
    Department: { type: 'UTF8' },
    'Company\t': { type: 'UTF8' },
    'Created At ': { type: 'TIMESTAMP_MILLIS' },
    'Profile Photo': { type: 'UTF8' },
    '  Description': { type: 'UTF8' },
    Activated: { type: 'BOOLEAN' },
  });
  const parquetname = directory + 'userdata.parquet';
  const writer = await parquet.ParquetWriter.openFile(schema, parquetname);
  for (const row of data) {
    await writer.appendRow(row);
  }
  await writer.close();
  console.log(`Wrote ${parquetname}`);
}

write();
