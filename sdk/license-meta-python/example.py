import safedep.messages.package.v1.license_meta_pb2 as license_meta_pb2
import json

from google.protobuf.json_format import MessageToDict

# Example to create a JSONL file with multiple
# LicenseMeta messages

# Full spec: https://buf.build/safedep/api/docs/main:safedep.messages.package.v1#safedep.messages.package.v1.LicenseMeta
def get_license_meta(id: str, name: str) -> license_meta_pb2.LicenseMeta:
    license_meta = license_meta_pb2.LicenseMeta()
    license_meta.license_id = id
    license_meta.name = name
    license_meta.reference_url = f'https://spdx.org/licenses/{id}.html'
    license_meta.details_url = f'https://spdx.org/licenses/{id}.json'

    # We don't have this info in example so we will just set it to false
    license_meta.osi_approved = False
    license_meta.saas_compatible = False
    license_meta.fsf_approved = False
    license_meta.commercial_use_allowed = False

    # Example compatibility fields
    compatibility = {
        'MIT': False,
        'GPL-3.0': False,
    }

    license_meta.compatibility.update(compatibility)

    return license_meta

def to_jsonl(license: license_meta_pb2.LicenseMeta) -> str:
    return json.dumps(MessageToDict(license, preserving_proto_field_name=True))

if __name__ == '__main__':
    licenses = [
        {
            'id': 'MIT',
            'name': 'MIT License'
        },
        {
            'id': 'APACHE-2.0',
            'name': 'Apache License 2.0'
        },
        {
            'id': 'GPL-3.0',
            'name': 'GNU General Public License v3.0'
        }
    ]

    for license in licenses:
        license_meta = get_license_meta(license['id'], license['name'])
        print(to_jsonl(license_meta))
