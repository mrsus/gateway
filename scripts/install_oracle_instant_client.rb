#!/usr/bin/env ruby

require 'rbconfig'
require 'fileutils'

INSTANT_CLIENT_DIR = ARGV[0]

def download_url
  "http://downloads.justapis.com/v5.1.0/oracle/#{os}_#{architecture}_instant_client_12_1.tar.gz"
end

def os
  @os ||= (
    host_os = RbConfig::CONFIG['host_os']
    case host_os
    when /mswin|msys|mingw|cygwin|bccwin|wince|emc/
      :windows
    when /darwin|mac os/
      :osx
    when /linux/
      :linux
    when /solaris|bsd/
      :unix
    else
      raise Error, "unknown os: #{host_os.inspect}"
    end
  )
end

def architecture
  1.size * 8
end

def create_config(template, new_first_line, destination_file)
  original_config = File.read(template)
  new_config = new_first_line
  new_config << original_config
  File.open(destination_file, 'w') {|f| f.puts new_config }
end

def do_install(extension)
 if File.symlink?(File.join(INSTANT_CLIENT_DIR,"libclntsh.#{extension}"))
    puts "Instant client appears to be already installed."
  else
    puts "Setting up instant client in #{INSTANT_CLIENT_DIR}"
    FileUtils::mkdir_p(INSTANT_CLIENT_DIR)
    `curl --silent #{download_url} | tar -zxv -C #{INSTANT_CLIENT_DIR} --strip 1`
    raise "Failed to download client from #{download_url} into #{INSTANT_CLIENT_DIR}" unless $?.success?
    prefix = "prefix=#{INSTANT_CLIENT_DIR}"
    oci8_pc = File.join(INSTANT_CLIENT_DIR,'oci8.pc')
    create_config(ARGV[1], prefix, oci8_pc)
    `cd #{INSTANT_CLIENT_DIR} && ln -s libclntsh.#{extension}.12.1 libclntsh.#{extension}`
    raise "Failed to create a symbolic link!" unless $?.success?
  end
end

if os == :osx
  do_install('dylib')
elsif os == :linux
  do_install('so')
else
  raise "Please implement me for #{os}!"
end