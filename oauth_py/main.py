from flask import Flask, request, redirect, url_for, session
from flask import render_template
from onelogin.saml2.auth import OneLogin_Saml2_Auth
from onelogin.saml2.idp_metadata_parser import OneLogin_Saml2_IdPMetadataParser

app = Flask(__name__)
# SAML settings
# Configure SAML settings using the parsed IdP settings
saml_auth = OneLogin_Saml2_Auth(request, app)

# Configure SAML settings - you need to replace these values with your IdP settings
saml_auth.idp = {
    "entityId": "http://www.okta.com/exk7cxugvlWY2Ao0C697",  # Your SP Entity ID (Issuer)
    "assertionConsumerService": {
        "url": "http://localhost:5000/sso",  # Your ACS URL
        "binding": "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
    },
    "NameIDFormat": "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified",
    "x509cert": """-----BEGIN CERTIFICATE-----
            MIIDqjCCApKgAwIBAgIGAYp/e6tDMA0GCSqGSIb3DQEBCwUAMIGVMQswCQYDVQQGEwJVUzETMBEG
            A1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzENMAsGA1UECgwET2t0YTEU
            MBIGA1UECwwLU1NPUHJvdmlkZXIxFjAUBgNVBAMMDXRyaWFsLTYyNDg4ODMxHDAaBgkqhkiG9w0B
            CQEWDWluZm9Ab2t0YS5jb20wHhcNMjMwOTEwMTQyNDI1WhcNMzMwOTEwMTQyNTI1WjCBlTELMAkG
            A1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWExFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xDTAL
            BgNVBAoMBE9rdGExFDASBgNVBAsMC1NTT1Byb3ZpZGVyMRYwFAYDVQQDDA10cmlhbC02MjQ4ODgz
            MRwwGgYJKoZIhvcNAQkBFg1pbmZvQG9rdGEuY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
            CgKCAQEAp8OBBhkmmyItGokMallc1HtxzLkTGYAX4qzMD0k8zw80jRfLLstzZL5Hv2CONMzx1F0S
            rhZsnUcmGkN8GQeonQIENUTzIH2gE0+YWgjiL/0yJx9kaCgRpJ0hoHrr7jGd8Y2ZRQgsVBqhquCT
            KM57t1o9KvKjl0bllrbDdyUo/Dw0CGzgor3+7e825I0rXoK9LhmjbZK6YamSRiNv29o1A2Ufxjlz
            PC2oP/l7P+opHZZS3ZyP6+lcOQPtY+ed2p0R0vvkTRhjv009Gb8J51BsT53BkJKhOX19U2yRWX3+
            QGNeH1LPbJTVe1zYEf4Y1I0zLhzL1UUm6zLM/C0EqJgUoQIDAQABMA0GCSqGSIb3DQEBCwUAA4IB
            AQCABpm6ABYBML3wTXsOKGHvjF9r/v3AiUhuxvSKjDd5gw9K9/4zmc0fn5Ir7r+kwZ44LfgACQz1
            yYGq3S4ANCsqFk6TGYd30GniVklYKeZ3b7UeJA7cy7tDw62O8x5JrvB5I/Vyab3ROrof+vckUaRU
            s/WUDzOOnejmmETFORmVEoOBZxjG/NdEalEyyfsdBOdxk/L2+TA9TJfq5vNGuJoCE3lcmIrl0Tlp
            jlZsHJ1dYaBSMR2bUv0vNVzT5oJHFejPLyzwBDufssefvBrdqs5MPaDOoxML4wG+YmzoA7/fdcp6
            SmSx2/lLXL4VdMZnjM7dr4FzsQ/a/92/wujWL4vS
            -----END CERTIFICATE-----
            """,
    "privateKey": "",
}


@app.route('/')
def index():
    if not session.get('samlUserdata'):
        return redirect(url_for('login'))
    return f"Hello, {session['samlUserdata']['email']}!"

@app.route('/login')
def login():
    return redirect(saml_auth.login())

@app.route('/sso', methods=['POST'])
def sso():
    saml_auth.process_response()
    errors = saml_auth.get_errors()
    if errors:
        return "SAML Authentication Error"
    if not saml_auth.is_authenticated():
        return "SAML Authentication failed"
    
    session['samlUserdata'] = saml_auth.get_attributes()
    return redirect(url_for('index'))

@app.route('/logout')
def logout():
    request_url = saml_auth.logout()
    return redirect(request_url)
from flask import jsonify

@app.route('/api/check-auth')
def check_auth():
    if 'samlUserdata' in session:
        return jsonify({'isAuthenticated': True, 'userData': session['samlUserdata']})
    return jsonify({'isAuthenticated': False})

@app.route('/slo', methods=['POST'])
def slo():
    request_id = None
    if 'SAMLRequest' in request.form:
        request_id = saml_auth.process_slo(request.form['SAMLRequest'])
    errors = saml_auth.get_errors()
    if len(errors) == 0:
        if request_id is not None:
            url = saml_auth.logout(request_id)
        else:
            url = saml_auth.logout()
        return redirect(url)
    return "SAML Single Logout failed"

if __name__ == '__main__':
    app.secret_key = 'your_secret_key'
    app.run(debug=True)