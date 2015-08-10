require File.expand_path('../lib/digitalocean_client/version', __FILE__)

Gem::Specification.new do |gem|
  gem.authors       = ['Sri Vishnu Totakura']
  gem.email         = ['t.srivishnu+digioceanclient@gmail.com']
  gem.description   = 'Used to connect to digital ocean and control your droplets'
  gem.summary       = 'DigitalOcean Control Client'
  gem.homepage      = 'https://github.com/tsrivishnu/digitalocean-client'
  gem.license       = 'MIT'

  gem.files         = `git ls-files`.split($OUTPUT_RECORD_SEPARATOR)
  gem.executables   = gem.files.grep(%r{^bin/}).map{ |f| File.basename(f) }
  # gem.test_files    = gem.files.grep(%r{^(test|spec|features)/})
  gem.name          = 'digitalocean_client'
  gem.require_paths = ['lib']
  gem.version       = DigitaloceanClient::VERSION

  gem.required_ruby_version = '>= 2.0'

  gem.add_development_dependency 'rspec', '>=3.0'
end
