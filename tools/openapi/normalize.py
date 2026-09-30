import json
import pathlib
import sys


spec_path = pathlib.Path(sys.argv[1])
spec = json.loads(spec_path.read_text())
if 'x-maco' in spec.get('info', {}):
    spec['x-maco'] = spec['info'].pop('x-maco')


def normalize_descriptions(value):
    if isinstance(value, dict):
        if 'x-maco-description' in value:
            value['description'] = value.pop('x-maco-description')
        for child in value.values():
            normalize_descriptions(child)
    elif isinstance(value, list):
        for child in value:
            normalize_descriptions(child)


normalize_descriptions(spec)
for methods in spec['paths'].values():
    for operation in methods.values():
        responses = operation.get('responses', {})
        for key in list(responses):
            if key.startswith('x-'):
                operation[key] = responses.pop(key)
        content = operation.get('requestBody', {}).get('content', {})
        form = content.get('application/x-www-form-urlencoded', {}).get('schema', {})
        if 'multipart/form-data' not in content or form.get('type') != 'file':
            continue
        field = form['title']
        content['multipart/form-data']['schema'] = {
            'type': 'object',
            'required': [field] if operation['requestBody'].get('required') else [],
            'properties': {field: {'type': 'string', 'format': 'binary'}},
        }
        del content['application/x-www-form-urlencoded']

spec_path.write_text(json.dumps(spec, indent=4) + '\n')
